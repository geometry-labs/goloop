package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/cmd/cli"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	DefaultKeyStorePass = "gochain"
)

type GoChainConfig struct {
	chain.Config
	P2PAddr       string `json:"p2p"`
	P2PListenAddr string `json:"p2p_listen"`
	EESocket      string `json:"ee_socket"`
	RPCAddr       string `json:"rpc_addr"`
	RPCDump       bool   `json:"rpc_dump"`
	RPCDebug      bool   `json:"rpc_debug"`
	EEInstances   int    `json:"ee_instances"`

	Key          []byte          `json:"key,omitempty"`
	KeyStoreData json.RawMessage `json:"key_store"`
	KeyStorePass string          `json:"key_password"`

	LogLevel     string               `json:"log_level"`
	ConsoleLevel string               `json:"console_level"`
	LogForwarder *log.ForwarderConfig `json:"log_forwarder,omitempty"`
}

func (config *GoChainConfig) String() string {
	return ""
}

func (config *GoChainConfig) Type() string {
	return "GoChainConfig"
}

func (config *GoChainConfig) Set(name string) error {
	config.FilePath, _ = filepath.Abs(name)
	if bs, e := ioutil.ReadFile(name); e == nil {
		if err := json.Unmarshal(bs, config); err != nil {
			return err
		}
	}
	return nil
}

var memProfileCnt int32 = 0

var (
	version = "unknown"
	build   = "unknown"
)

var genesisStorage, genesisPath string
var keyStoreFile, keyStoreSecret string
var saveFile, saveKeyStore string
var cfg GoChainConfig
var cpuProfile, memProfile string
var chainDir string
var eeSocket string
var modLevels map[string]string
var lfCfg log.ForwarderConfig

func main() {
	cmd := &cobra.Command{
		Use:  os.Args[0],
		Args: cobra.ExactArgs(0),
	}
	flag := cmd.PersistentFlags()

	flag.VarP(&cfg, "config", "c", "Configuration file path")
	flag.StringVar(&saveFile, "save", "", "File path for storing current configuration(it exits after save)")
	flag.StringVar(&saveKeyStore, "save_key_store", "", "File path for storing current KeyStore")
	flag.StringVar(&cfg.Channel, "channel", "default", "Channel name for the chain")
	flag.StringVar(&cfg.P2PAddr, "p2p", "127.0.0.1:8080", "Advertise ip-port of P2P")
	flag.StringVar(&cfg.P2PListenAddr, "p2p_listen", "", "Listen ip-port of P2P")
	flag.IntVar(&cfg.NID, "nid", 0, "Chain Network ID")
	flag.StringVar(&cfg.RPCAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	flag.BoolVar(&cfg.RPCDump, "rpc_dump", false, "JSON-RPC Request, Response Dump flag")
	flag.BoolVar(&cfg.RPCDebug, "rpc_debug", false, "JSON-RPC Debug enable")
	flag.StringVar(&cfg.SeedAddr, "seed", "", "Ip-port of Seed")
	flag.StringVar(&genesisStorage, "genesis_storage", "", "Genesis storage path")
	flag.StringVar(&genesisPath, "genesis", "", "Genesis template directory or file")
	flag.StringVar(&cfg.DBType, "db_type", "goleveldb", "Name of database system(badgerdb, *goleveldb, boltdb, mapdb)")
	flag.UintVar(&cfg.Role, "role", 2, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	flag.StringVarP(&eeSocket, "ee_socket", "s", "", "Execution engine socket path(default:.chain/<address>/ee.sock")
	flag.StringVar(&keyStoreFile, "key_store", "", "KeyStore file for wallet")
	flag.StringVar(&keyStoreSecret, "key_secret", "", "Secret(password) file for KeyStore")
	flag.StringVar(&cfg.KeyStorePass, "key_password", "", "Password for the KeyStore file")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "CPU Profiling data file")
	flag.StringVar(&memProfile, "memprofile", "", "Memory Profiling data file")
	flag.StringVar(&chainDir, "chain_dir", "", "Chain data directory(default:.chain/<address>/<nid>")
	flag.IntVar(&cfg.EEInstances, "ee_instances", 1, "Number of execution engines")
	flag.IntVar(&cfg.ConcurrencyLevel, "concurrency", 1, "Maximum number of executors to use for concurrency")
	flag.IntVar(&cfg.NormalTxPoolSize, "normal_tx_pool", 0, "Normal transaction pool size")
	flag.IntVar(&cfg.PatchTxPoolSize, "patch_tx_pool", 0, "Patch transaction pool size")
	flag.IntVar(&cfg.MaxBlockTxBytes, "max_block_tx_bytes", 0, "Maximum size of ransactions in a block")
	flag.StringVar(&cfg.NodeCache, "node_cache", chain.NodeCacheDefault, "Node cache(none,small,large)")
	flag.StringVar(&cfg.LogLevel, "log_level", "debug", "Main log level")
	flag.StringVar(&cfg.ConsoleLevel, "console_level", "trace", "Console log level")
	flag.StringToStringVar(&modLevels, "mod_level", nil, "Console log level for specific module (<mod>=<level>,...)")
	flag.StringVar(&lfCfg.Vendor, "log_forwarder_vendor", "", "LogForwarder vendor (fluentd,logstash)")
	flag.StringVar(&lfCfg.Address, "log_forwarder_address", "", "LogForwarder address")
	flag.StringVar(&lfCfg.Level, "log_forwarder_level", "info", "LogForwarder level")
	flag.StringVar(&lfCfg.Name, "log_forwarder_name", "", "LogForwarder name")
	flag.StringToString("log_forwarder_options", nil, "LogForwarder options, comma-separated 'key=value'")

	cmd.Run = Execute
	cmd.Execute()
}

func Execute(cmd *cobra.Command, args []string) {

	if len(keyStoreFile) > 0 {
		if ks, err := ioutil.ReadFile(keyStoreFile); err != nil {
			log.Panicf("Fail to open KeyStore file=%s err=%+v", keyStoreFile, err)
		} else {
			cfg.KeyStoreData = ks
			cfg.Key = []byte{}
		}
	}

	keyStorePass := []byte(cfg.KeyStorePass)
	if len(keyStoreSecret) > 0 {
		if ks, err := ioutil.ReadFile(keyStoreSecret); err != nil {
			log.Panicf("Fail to open KeySecret file=%s err=%+v", keyStoreSecret, err)
		} else {
			keyStorePass = ks
		}
	}

	var priK *crypto.PrivateKey
	if len(cfg.Key) > 0 {
		var err error
		if priK, err = crypto.ParsePrivateKey(cfg.Key); err != nil {
			log.Panicf("Illegal key data=[%x]", cfg.Key)
		}
		cfg.Key = nil
	}

	if len(cfg.KeyStoreData) > 0 {
		var err error
		if len(keyStorePass) == 0 {
			log.Panicf("There is no password information for the KeyStore")
		}
		priK, err = wallet.DecryptKeyStore(cfg.KeyStoreData, keyStorePass)
		if err != nil {
			log.Panicf("Fail to decrypt KeyStore err=%+v", err)
		}
	} else {
		// make sure that cfg.KeyStoreData always has valid value to let them
		// be stored with -save_key_store option even though the key is
		// provided by cfg.Key value.
		if priK == nil {
			priK, _ = crypto.GenerateKeyPair()
		}
		if len(keyStorePass) == 0 {
			cfg.KeyStorePass = DefaultKeyStorePass
			keyStorePass = []byte(cfg.KeyStorePass)
		}

		if ks, err := wallet.EncryptKeyAsKeyStore(priK, keyStorePass); err != nil {
			log.Panicf("Fail to encrypt private key err=%+v", err)
		} else {
			cfg.KeyStoreData = ks
		}
	}
	wallet, _ := wallet.NewFromPrivateKey(priK)

	if len(genesisStorage) > 0 {
		storage, err := ioutil.ReadFile(genesisStorage)
		if err != nil {
			log.Panicf("Fail to open genesisStorage=%s err=%+v\n", genesisStorage, err)
		}
		cfg.GenesisStorage, err = gs.New(storage)
		if err != nil {
			log.Panicf("Failed to load genesisStorage\n")
		}
	} else if len(genesisPath) > 0 {
		storage := bytes.NewBuffer(nil)
		if err := gs.WriteFromPath(storage, genesisPath); err != nil {
			log.Printf("FAIL to generate gs. err = %s, path = %s\n", err, genesisPath)
		}
		var err error
		cfg.GenesisStorage, err = gs.New(storage.Bytes())
		if err != nil {
			log.Panicf("Failed to load genesisStorage\n")
		}
	} else if len(cfg.Genesis) == 0 {
		genesis := map[string]interface{}{
			"accounts": []map[string]interface{}{
				{
					"name":    "god",
					"address": wallet.Address().String(),
					"balance": "0x2961fff8ca4a62327800000",
				},
				{
					"name":    "treasury",
					"address": "hx1000000000000000000000000000000000000000",
					"balance": "0x0",
				},
			},
			"chain": map[string]interface{}{
				"validatorList": []string{
					wallet.Address().String(),
				},
			},
			"message": "gochain generated genesis",
		}
		if cfg.NID != 0 {
			genesis["nid"] = fmt.Sprintf("%#x", cfg.NID)
		}
		cfg.Genesis, _ = json.Marshal(genesis)
	}

	if cfg.NID == 0 {
		gtx, _ := transaction.NewGenesisTransaction(cfg.Genesis)
		cfg.NID = gtx.NID()
	}

	if len(saveKeyStore) > 0 {
		ks := bytes.NewBuffer(nil)
		if err := json.Indent(ks, cfg.KeyStoreData, "", "  "); err != nil {
			log.Panicf("Fail to indenting key data err=%+v", err)
		}
		if err := ioutil.WriteFile(saveKeyStore, ks.Bytes(), 0600); err != nil {
			log.Panicf("Fail to save key store to the file=%s err=%+v", saveKeyStore, err)
		}
	}

	var tLfCfg log.ForwarderConfig
	if cfg.LogForwarder != nil {
		tLfCfg = *cfg.LogForwarder
	}
	if lfCfg.Vendor == "" {
		lfCfg.Vendor = tLfCfg.Vendor
	}
	if lfCfg.Address == "" {
		lfCfg.Address = tLfCfg.Address
	}
	if lfCfg.Level == "" {
		lfCfg.Level = tLfCfg.Level
	}
	if lfCfg.Name == "" {
		lfCfg.Name = tLfCfg.Name
	}
	if lfOpts, err := cli.GetStringMap(cmd.Flag("log_forwarder_options")); err != nil {
		log.Panicf("Failed to parse LogForwarderOptions\n")
	} else {
		lfCfg.Options = lfOpts
	}
	if len(tLfCfg.Options) > 0 {
		if lfCfg.Options == nil {
			lfCfg.Options = tLfCfg.Options
		} else {
			for k, v := range tLfCfg.Options {
				if _, ok := lfCfg.Options[k]; !ok {
					lfCfg.Options[k] = v
				}
			}
		}
	}
	if lfCfg.Vendor != "" {
		cfg.LogForwarder = &lfCfg
	}

	if saveFile != "" {
		f, err := os.OpenFile(saveFile,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Panicf("Fail to open file=%s err=%+v", saveFile, err)
		}

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&cfg); err != nil {
			log.Panicf("Fail to generate JSON for %+v", cfg)
		}
		f.Close()
		os.Exit(0)
	}

	logger := log.WithFields(log.Fields{
		log.FieldKeyWallet: hex.EncodeToString(wallet.Address().ID()),
	})
	log.SetGlobalLogger(logger)
	stdlog.SetOutput(logger.WriterLevel(log.WarnLevel))

	if lv, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.Panicf("Fail to parse loglevel level=%s", cfg.LogLevel)
	} else {
		logger.SetLevel(lv)
	}

	if lv, err := log.ParseLevel(cfg.ConsoleLevel); err != nil {
		log.Panicf("Fail to parse loglevel level=%s", cfg.ConsoleLevel)
	} else {
		logger.SetConsoleLevel(lv)
	}

	if len(modLevels) > 0 {
		for mod, lvString := range modLevels {
			if lv, err := log.ParseLevel(lvString); err != nil {
				log.Panicf("Log level(%s) for %s isn't valid err=%+v",
					lvString, mod, err)
			} else {
				logger.SetModuleLevel(mod, lv)
			}
		}
	}

	if cfg.LogForwarder != nil {
		if err := log.AddForwarder(cfg.LogForwarder); err != nil {
			log.Fatalf("Invalid log_forwarder err:%+v", err)
		}
	}

	if chainDir != "" {
		cfg.BaseDir = cfg.ResolveRelative(chainDir)
	}

	if cfg.BaseDir == "" {
		cfg.BaseDir = cfg.ResolveRelative(path.Join(".chain",
			wallet.Address().String(), strconv.FormatInt(int64(cfg.NID), 16)))
	}

	if eeSocket != "" {
		cfg.EESocket = cfg.ResolveRelative(eeSocket)
	}

	if cfg.EESocket == "" {
		cfg.EESocket = cfg.ResolveRelative(path.Join(".chain",
			wallet.Address().String(), "ee.sock"))
	}

	if cpuProfile != "" {
		if err := cli.StartCPUProfile(cpuProfile); err != nil {
			log.Panicf("Fail to start cpu profiling err=%+v", err)
		}
	}

	if memProfile != "" {
		if err := cli.StartMemoryProfile(memProfile); err != nil {
			log.Panicf("Fail to start memory profiling err=%+v", err)
		}
	}

	logoLines := []string{
		"  ____  ___   ____ _   _    _    ___ _   _ ",
		" / ___|/ _ \\ / ___| | | |  / \\  |_ _| \\ | |",
		"| |  _| | | | |   | |_| | / _ \\  | ||  \\| |",
		"| |_| | |_| | |___|  _  |/ ___ \\ | || |\\  |",
		" \\____|\\___/ \\____|_| |_/_/   \\_\\___|_| \\_|",
	}
	for _, l := range logoLines {
		log.Infoln(l)
	}
	log.Infof("Version : %s", version)
	log.Infof("Build   : %s", build)

	metric.Initialize(wallet)
	nt := network.NewTransport(cfg.P2PAddr, wallet, logger)
	if cfg.P2PListenAddr != "" {
		_ = nt.SetListenAddress(cfg.P2PListenAddr)
	}
	err := nt.Listen()
	if err != nil {
		log.Panicf("FAIL to listen P2P err=%+v", err)
	}
	defer nt.Close()

	ee, err := eeproxy.NewPythonEE(logger)
	if err != nil {
		log.Panicf("FAIL to create PythonEE err=%+v", err)
	}

	pm, err := eeproxy.NewManager("unix", cfg.EESocket, logger, ee)
	if err != nil {
		log.Panicln("FAIL to start EEManager")
	}
	go pm.Loop()

	pm.SetInstances(cfg.EEInstances, cfg.EEInstances, cfg.EEInstances)

	// TODO : server-chain setting
	srv := server.NewManager(cfg.RPCAddr, cfg.RPCDump, cfg.RPCDebug, "", wallet, logger)
	hex.EncodeToString(wallet.Address().ID())
	c := chain.NewChain(wallet, nt, srv, pm, logger, &cfg.Config)
	err = c.Init(true)
	if err != nil {
		log.Panicf("FAIL to initialize Chain err=%+v", err)
	}
	err = c.Start(true)
	if err != nil {
		log.Panicf("FAIL to start Chain err=%+v", err)
	}

	// main loop
	srv.Start()
}
