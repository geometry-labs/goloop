package icstate

import (
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/service/scoredb"
)

var Schema = func(prefix int, key string) (schema containerdb.KeyBuilder) {
	builder := containerdb.HashBuilder

	switch prefix {
	case -1:
		schema = containerdb.ToKey(builder, key)
	case 0:
		schema = containerdb.ToKey(builder, scoredb.ArrayDBPrefix, key)
	case 1:
		schema = containerdb.ToKey(builder, scoredb.DictDBPrefix, key)
	case 2:
		schema = containerdb.ToKey(builder, scoredb.VarDBPrefix, key)
	}
	return
}

const (
	VarIRep                                  = "irep"
	VarRRep                                  = "rrep"
	VarMainPRepCount                         = "main_prep_count"
	VarSubPRepCount                          = "sub_prep_count"
	VarTotalStake                            = "total_stake"
	VarIISSVersion                           = "iiss_version"
	VarTermPeriod                            = "term_period"
	VarBondRequirement                       = "bond_requirement"
	VarUnbondingPeriodMultiplier             = "unbonding_period_multiplier"
	VarLockMinMultiplier                     = "lockMinMultiplier"
	VarLockMaxMultiplier                     = "lockMaxMultiplier"
	VarRewardFund                            = "reward_fund"
	VarUnbondingMax                          = "unbonding_max"
	VarValidationPenaltyCondition            = "validation_penalty_condition"
	VarConsistentValidationPenaltyCondition  = "consistent_validation_penalty_condition"
	VarConsistentValidationPenaltyMask       = "consistent_validation_penalty_mask"
	VarConsistentValidationPenaltySlashRatio = "consistent_validation_penalty_slashRatio"
)

var (
	// Keys schema
	IssueKey          = Schema(-1, "issue_icx").Build()
	RewardCalcInfoKey = Schema(-1, "reward_calc_info").Build()

	// DictDB schema
	AccountDictPrefix        = Schema(0, "account_db")
	prepBaseDictPrefix       = Schema(0, "prep_base")
	prepStatusDictPrefix     = Schema(0, "prep_status")
	nodeOwnerDictPrefix      = Schema(0, "node_owner")
	unstakingTimerDictPrefix = Schema(0, "timer_unstaking")
	unbondingTimerDictPrefix = Schema(0, "timer_unbonding")

	// ArrayDB schema
	LastValidatorsPrefix  = Schema(1, "last_validators")
	activePRepArrayPrefix = Schema(1, "active_prep")

	// VarDB schema
	termVarPrefix        = Schema(2, "term")
	UnstakeSlotMaxPrefix = Schema(2, "unstake_slot_max")
)
