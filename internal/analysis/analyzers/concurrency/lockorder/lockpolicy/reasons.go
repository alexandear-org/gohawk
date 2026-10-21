package lockpolicy

// Reason explains lock identity, release, and ordering evidence.
type Reason uint8

const (
	None Reason = iota
	CarriedConstantBranchInfeasible
	ConditionalCallerReleaseProven
	ConditionalCallerReleaseUnknown
	CrossOwnerClassUnknown
	DeferredReleaseProven
	CycleOrderRecorded
	ExclusiveObjectBeforePublication
	ExclusiveParameterFromFreshCallers
	FreshBoundOwnerIdentityUnknown
	FreshFieldIdentityUnknown
	ImportedWriterGuardUnknown
	LoadedAcquisitionGuardUnknown
	LockStateBudgetExhausted
	HelperReleaseUnproven
	HeldForCallerProven
	HeldForCallerUnknown
	MutexActionObserved
	ReleaseIdentityUnknown
	NoFreshBoundOwner
	NoFreshFieldWitness
	OppositeOrderRecorded
	PredecessorConstantBranchInfeasible
	RepeatedConditionInfeasible
	StableParameterBranchInfeasible
	PrivateMutexOnly
	ReleaseOwnershipUnknown
	UnreleasedReturn
	ReadLockWrite
	ExclusiveWriterUnknown
	PrivateWriteStorage
	FieldGuardUnknown
	InitialPublicationUnknown
	ExclusiveOwnershipUnknown
	Count
)

var reasonCodes = [...]string{
	ExclusiveOwnershipUnknown:           "exclusive-ownership-unknown",
	InitialPublicationUnknown:           "initial-publication-order-unknown",
	FieldGuardUnknown:                   "field-guard-unknown",
	PrivateWriteStorage:                 "private-write-storage",
	PrivateMutexOnly:                    "private-mutex-only",
	ReleaseOwnershipUnknown:             "release-ownership-unknown",
	UnreleasedReturn:                    "unreleased-return",
	ReadLockWrite:                       "read-lock-write",
	ExclusiveWriterUnknown:              "exclusive-writer-guard-unknown",
	None:                                "",
	CarriedConstantBranchInfeasible:     "carried-constant-branch-infeasible",
	ConditionalCallerReleaseProven:      "conditional-caller-release-proven",
	ConditionalCallerReleaseUnknown:     "conditional-caller-release-unknown",
	CrossOwnerClassUnknown:              "cross-owner-class-unknown",
	DeferredReleaseProven:               "deferred-release-proven",
	CycleOrderRecorded:                  "cycle-order-recorded",
	ExclusiveObjectBeforePublication:    "exclusive-object-before-publication",
	ExclusiveParameterFromFreshCallers:  "exclusive-parameter-from-fresh-callers",
	FreshBoundOwnerIdentityUnknown:      "fresh-bound-owner-identity-unknown",
	FreshFieldIdentityUnknown:           "fresh-field-identity-unknown",
	ImportedWriterGuardUnknown:          "imported-writer-guard-unknown",
	LoadedAcquisitionGuardUnknown:       "loaded-acquisition-guard-unknown",
	LockStateBudgetExhausted:            "lock-state-budget-exhausted",
	HelperReleaseUnproven:               "helper-release-unproven",
	HeldForCallerProven:                 "held-for-caller-proven",
	HeldForCallerUnknown:                "held-for-caller-unknown",
	MutexActionObserved:                 "mutex-action-observed",
	ReleaseIdentityUnknown:              "release-identity-unknown",
	NoFreshBoundOwner:                   "no-fresh-bound-owner",
	NoFreshFieldWitness:                 "no-fresh-field-witness",
	OppositeOrderRecorded:               "opposite-order-recorded",
	PredecessorConstantBranchInfeasible: "predecessor-constant-branch-infeasible",
	RepeatedConditionInfeasible:         "repeated-condition-infeasible",
	StableParameterBranchInfeasible:     "stable-parameter-branch-infeasible",
}

func (reason Reason) String() string {
	if int(reason) >= len(reasonCodes) {
		return "invalid-lock-reason"
	}
	return reasonCodes[reason]
}
