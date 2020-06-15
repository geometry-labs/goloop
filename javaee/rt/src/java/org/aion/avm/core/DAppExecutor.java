package org.aion.avm.core;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.CodedException;
import foundation.icon.ee.types.DAppRuntimeState;
import foundation.icon.ee.types.ObjectGraph;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.Transaction;
import i.AvmError;
import i.AvmException;
import i.GenericCodedException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.InstrumentationHelpers;
import i.InternedClasses;
import i.JvmError;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.parallel.TransactionTask;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class DAppExecutor {
    private static final Logger logger = LoggerFactory.getLogger(DAppExecutor.class);

    public static Result call(IExternalState externalState,
                              LoadedDApp dapp,
                              ReentrantDAppStack.ReentrantState stateToResume,
                              TransactionTask task,
                              Address senderAddress,
                              Address dappAddress,
                              Transaction tx,
                              AvmConfiguration conf) throws AvmError {
        Result result = null;

        // Note that the instrumentation is just a per-thread access to the state stack - we can grab it at any time as it never changes for this thread.
        IInstrumentation threadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        
        // We need to get the interned classes before load the graph since it might need to instantiate class references.
        InternedClasses initialClassWrappers = dapp.getInternedClasses();

        var saveItem = task.getReentrantDAppStack().getSaveItem(dappAddress);
        DAppRuntimeState prevRS;
        if (saveItem == null) {
            var raw = externalState.getObjectGraph(dappAddress);
            var graph = ObjectGraph.getInstance(raw);
            prevRS = new DAppRuntimeState(null, graph);
        } else {
            prevRS = saveItem.getRuntimeState();
        }
        var nextHashCode = dapp.loadRuntimeState(prevRS);

        // Used for deserialization billing
        int rawGraphDataLength = prevRS.getGraph().getGraphData().length + 4;

        // Note that we need to store the state of this invocation on the reentrant stack in case there is another call into the same app.
        // This is required so that the call() mechanism can access it to save/reload its ContractEnvironmentState and so that the underlying
        // instance loader (ReentrantGraphProcessor/ReflectionStructureCodec) can be notified when it becomes active/inactive (since it needs
        // to know if it is loading an instance
        ReentrantDAppStack.ReentrantState thisState = new ReentrantDAppStack.ReentrantState(dappAddress, dapp, nextHashCode);
        var prevState = task.getReentrantDAppStack().getTop();
        task.getReentrantDAppStack().pushState(thisState);

        IBlockchainRuntime br = new BlockchainRuntimeImpl(externalState,
                                                          task,
                                                          senderAddress,
                                                          dappAddress,
                                                          tx,
                                                          dapp.runtimeSetup,
                                                          dapp,
                                                          conf.enableContextPrintln);
        FrameContextImpl fc = new FrameContextImpl(externalState);
        InstrumentationHelpers.pushNewStackFrame(dapp.runtimeSetup, dapp.loader, tx.getLimit(), nextHashCode, initialClassWrappers, fc);
        IBlockchainRuntime previousRuntime = dapp.attachBlockchainRuntime(br);

        try {
            // It is now safe for us to bill for the cost of loading the graph (the cost is the same, whether this came from the caller or the disk).
            // (note that we do this under the try since aborts can happen here)
            threadInstrumentation.chargeEnergy(StorageFees.READ_PRICE_PER_BYTE * rawGraphDataLength);

            // Call the main within the DApp.
            Object ret;
            try {
                ret = dapp.callMethod(tx.getMethod(), tx.getParams());
            } finally {
                externalState.waitForCallbacks();
            }

            var newRS = dapp.saveRuntimeState();

            if (externalState.isReadOnly() && !prevRS.isAcceptableChangeInReadOnly(newRS)) {
                throw new GenericCodedException(Status.AccessDenied);
            }

            // Save back the state before we return.
            if (null == stateToResume) {
                byte[] postCallGraphData = newRS.getGraph().getRawData();
                // Bill for writing this size.
                threadInstrumentation.chargeEnergy(StorageFees.WRITE_PRICE_PER_BYTE * postCallGraphData.length);
                externalState.putObjectGraph(dappAddress, postCallGraphData);
            }

            long energyUsed = tx.getLimit() - threadInstrumentation.energyLeft();
            result = new Result(Status.Success, energyUsed, ret);
            if (prevState != null) {
                prevState.getSaveItems().putAll(thisState.getSaveItems());
                prevState.getSaveItems().put(dappAddress, new ReentrantDAppStack.SaveItem(dapp, newRS));
            }
        } catch (AvmException e) {
            if (conf.enableVerboseContractErrors) {
                System.err.println("DApp invocation failed : " + e.getMessage());
                e.printStackTrace();
            }
            int code = Status.UnknownFailure;
            String msg = null;
            if (e instanceof CodedException) {
                code = ((CodedException) e).getCode();
                msg = e.getMessage();
            }
            if (msg == null) {
                msg = Status.getMessage(code);
            }
            long stepUsed = tx.getLimit() - threadInstrumentation.energyLeft();
            return new Result(code, stepUsed, msg);
        } finally {
            // Once we are done running this, no matter how it ended, we want to detach our thread from the DApp.
            InstrumentationHelpers.popExistingStackFrame(dapp.runtimeSetup);
            // This state was only here while we were running, in case someone else needed to change it so now we can pop it.
            task.getReentrantDAppStack().popState();

            // Re-attach the previously detached IBlockchainRuntime instance.
            dapp.attachBlockchainRuntime(previousRuntime);
        }
        return result;
    }
}
