package lockorder

import "github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/lockorder/lockpolicy"

// lockReason retains the analyzer's policy and trace vocabulary.
type lockReason = lockpolicy.Reason

const (
	lockReasonNone                                = lockpolicy.None
	lockReasonCarriedConstantBranchInfeasible     = lockpolicy.CarriedConstantBranchInfeasible
	lockReasonConditionalCallerReleaseProven      = lockpolicy.ConditionalCallerReleaseProven
	lockReasonConditionalCallerReleaseUnknown     = lockpolicy.ConditionalCallerReleaseUnknown
	lockReasonCrossOwnerClassUnknown              = lockpolicy.CrossOwnerClassUnknown
	lockReasonDeferredReleaseProven               = lockpolicy.DeferredReleaseProven
	lockReasonCycleOrderRecorded                  = lockpolicy.CycleOrderRecorded
	lockReasonExclusiveObjectBeforePublication    = lockpolicy.ExclusiveObjectBeforePublication
	lockReasonExclusiveParameterFromFreshCallers  = lockpolicy.ExclusiveParameterFromFreshCallers
	lockReasonFreshBoundOwnerIdentityUnknown      = lockpolicy.FreshBoundOwnerIdentityUnknown
	lockReasonFreshFieldIdentityUnknown           = lockpolicy.FreshFieldIdentityUnknown
	lockReasonImportedWriterGuardUnknown          = lockpolicy.ImportedWriterGuardUnknown
	lockReasonLoadedAcquisitionGuardUnknown       = lockpolicy.LoadedAcquisitionGuardUnknown
	lockReasonLockStateBudgetExhausted            = lockpolicy.LockStateBudgetExhausted
	lockReasonHelperReleaseUnproven               = lockpolicy.HelperReleaseUnproven
	lockReasonHeldForCallerProven                 = lockpolicy.HeldForCallerProven
	lockReasonHeldForCallerUnknown                = lockpolicy.HeldForCallerUnknown
	lockReasonMutexActionObserved                 = lockpolicy.MutexActionObserved
	lockReasonReleaseIdentityUnknown              = lockpolicy.ReleaseIdentityUnknown
	lockReasonNoFreshBoundOwner                   = lockpolicy.NoFreshBoundOwner
	lockReasonNoFreshFieldWitness                 = lockpolicy.NoFreshFieldWitness
	lockReasonOppositeOrderRecorded               = lockpolicy.OppositeOrderRecorded
	lockReasonPredecessorConstantBranchInfeasible = lockpolicy.PredecessorConstantBranchInfeasible
	lockReasonRepeatedConditionInfeasible         = lockpolicy.RepeatedConditionInfeasible
	lockReasonStableParameterBranchInfeasible     = lockpolicy.StableParameterBranchInfeasible
	lockReasonPrivateMutexOnly                    = lockpolicy.PrivateMutexOnly
	lockReasonReleaseOwnershipUnknown             = lockpolicy.ReleaseOwnershipUnknown
	lockReasonUnreleasedReturn                    = lockpolicy.UnreleasedReturn
	lockReasonReadLockWrite                       = lockpolicy.ReadLockWrite
	lockReasonExclusiveWriterUnknown              = lockpolicy.ExclusiveWriterUnknown
	lockReasonPrivateWriteStorage                 = lockpolicy.PrivateWriteStorage
	lockReasonFieldGuardUnknown                   = lockpolicy.FieldGuardUnknown
	lockReasonInitialPublicationUnknown           = lockpolicy.InitialPublicationUnknown
	lockReasonExclusiveOwnershipUnknown           = lockpolicy.ExclusiveOwnershipUnknown
	lockReasonCount                               = lockpolicy.Count
)
