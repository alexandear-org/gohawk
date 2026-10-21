package resourcepolicy

// Reason names stable resource lifetime evidence for policy and tracing.
type Reason uint8

const (
	None Reason = iota
	AcquisitionErrorProven
	AggregateOwnerMayEscape
	AmbiguousCleanupValue
	AmbiguousHelperCleanupValue
	Appended
	BudgetExhausted
	CallEffectsAsynchronousExposure
	CallResultMayTransfer
	DirectMayCarry
	AggregateMayCarry
	WrapperMayCarry
	ParentCleanup
	CapturedAggregateOwner
	CapturedBodyGuardedCleanup
	CapturedByLiteralCallingUnreadableCallee
	CapturedByPossiblyRetainedCallback
	CapturedByPriorCleanup
	CapturedByRetainingLiteral
	CapturedByStartedLiteral
	CapturedCellMayCleanup
	CompressionOutputMayBeAbandoned
	CanceledAcquisition
	DynamicCallee
	ErrorPredicateFalseForNil
	ErrorTypeAssertionSucceeded
	ErrorsAsExactAcquisitionError
	ErrorsIsNonNilFilesystemSentinel
	ErrorsIsNonNilContextSentinel
	EvidenceNotFound
	EvidenceUnavailable
	ExactErrorEqualsNonNilFilesystemSentinel
	HeadAcquisition
	HeadClientNotUnconfigured
	HeadRequestModified
	HelperCleanupInLoop
	ImportedHelperCleanupInLoop
	InterfaceMethod
	HeaderOnlyAcquisition
	LocalServerClientOverrideUnresolved
	LocalServerHandlerUnavailable
	LocalServerHeaderOnlyEffects
	LocalServerIdentityUnavailable
	LocalServerWriterEffectsUnavailable
	NestedInTransferredArgument
	Untouched
	OpaqueConsumption
	OptionalAcquisitionSuccessPhi
	PairedErrorHelperCleanup
	PriorDeferMayCleanCapturedCell
	ReleaseProven
	RepeatedGuardEdgeUnknown
	ResourceReturnPath
	ReturnedCleanupProjection
	ReturnedMayTransfer
	ReturnedWrapperRetains
	ReturnedProjectionLacksCleanup
	ReturnedViewCannotRelease
	RowsTransactionFinished
	TransactionContextCanceled
	SentToChannel
	ResponseBodyAggregateHandoff
	Settled
	StatementParentClosed
	StoredInMap
	StoredOnCollectionOwner
	IndirectDestinationUnknown
	WrapperStoredOnForeignOwner
	AcquisitionUnreachable
	TestifyNoErrorGuard
	UnownedReturn
	UnsummarizedCallee
	OSIsNotExist
	OSIsExist
	OSIsPermission
	OSIsTimeout
	ProcessExitReclaims
	ReturnedRetainingWrapper
	AppendedToLocalCollection
	CollectionReturned
	CollectionReleasedByHelper
	CollectionReleased
	CollectionUseUnknown
	RowsExhaustedEdgeUnknown
	ResultGuardedDefer
	ResultGuardedRelease
	ResultGuardedUnknown
	MemoryWriter
	AcquisitionLocationUnknown
	PriorDeferMayRelease
	Count
)

// These codes are a trace compatibility boundary, never proof inputs. In
// particular, unset ("") and an inspected-but-untouched instruction ("none")
// stay distinct. Numeric ordinals can change without changing that contract.
var reasonCodes = [...]string{
	DirectMayCarry:                           "direct-value-may-carry-resource",
	AggregateMayCarry:                        "aggregate-may-carry-resource",
	WrapperMayCarry:                          "wrapper-may-carry-resource",
	MemoryWriter:                             "memory-writer-no-external-resource",
	AcquisitionLocationUnknown:               "acquisition-location-unknown",
	PriorDeferMayRelease:                     "prior-defer-may-release",
	None:                                     "",
	AcquisitionErrorProven:                   "acquisition-error-proven",
	AggregateOwnerMayEscape:                  "aggregate-owner-may-escape",
	AmbiguousCleanupValue:                    "ambiguous-cleanup-value",
	AmbiguousHelperCleanupValue:              "ambiguous-helper-cleanup-value",
	Appended:                                 "appended",
	AppendedToLocalCollection:                "appended-to-local-collection",
	CollectionReturned:                       "collection-returned",
	CollectionReleasedByHelper:               "collection-released-by-helper",
	CollectionReleased:                       "collection-released",
	CollectionUseUnknown:                     "collection-use-unknown",
	RowsExhaustedEdgeUnknown:                 "rows-exhausted-edge-unknown",
	ResultGuardedDefer:                       "result-guarded-defer",
	ResultGuardedRelease:                     "result-guarded-release",
	ResultGuardedUnknown:                     "result-guarded-unknown",
	BudgetExhausted:                          "budget-exhausted",
	CallEffectsAsynchronousExposure:          "call-effects-asynchronous-exposure",
	CallResultMayTransfer:                    "call-result-may-transfer",
	ParentCleanup:                            "caller-owned-database-cleanup",
	CapturedAggregateOwner:                   "captured-aggregate-owner",
	CapturedBodyGuardedCleanup:               "captured-body-guarded-cleanup",
	CapturedByLiteralCallingUnreadableCallee: "captured-by-literal-calling-unreadable-callee",
	CapturedByPossiblyRetainedCallback:       "captured-by-possibly-retained-callback",
	CapturedByPriorCleanup:                   "captured-by-prior-cleanup",
	CapturedByRetainingLiteral:               "captured-by-retaining-literal",
	CapturedByStartedLiteral:                 "captured-by-started-literal",
	CapturedCellMayCleanup:                   "captured-cell-may-cleanup",
	CompressionOutputMayBeAbandoned:          "compression-output-may-be-abandoned",
	CanceledAcquisition:                      "context-canceled-before-acquisition",
	DynamicCallee:                            "dynamic-callee",
	ErrorPredicateFalseForNil:                "error-predicate-false-for-nil",
	ErrorTypeAssertionSucceeded:              "error-type-assertion-succeeded",
	ErrorsAsExactAcquisitionError:            "errors-as-exact-acquisition-error",
	ErrorsIsNonNilContextSentinel:            "errors-is-non-nil-context-sentinel",
	ErrorsIsNonNilFilesystemSentinel:         "errors-is-non-nil-filesystem-sentinel",
	EvidenceNotFound:                         "evidence-not-found",
	EvidenceUnavailable:                      "evidence-unavailable",
	ExactErrorEqualsNonNilFilesystemSentinel: "exact-error-equals-non-nil-filesystem-sentinel",
	HeadAcquisition:                          "head-body-acquisition-uncertain",
	HeadClientNotUnconfigured:                "head-client-not-unconfigured",
	HeadRequestModified:                      "head-request-modified",
	HelperCleanupInLoop:                      "helper-cleanup-in-loop",
	ImportedHelperCleanupInLoop:              "imported-helper-cleanup-in-loop",
	InterfaceMethod:                          "interface-method",
	HeaderOnlyAcquisition:                    "local-header-only-body-uncertain",
	LocalServerClientOverrideUnresolved:      "local-server-client-override-unresolved",
	LocalServerHandlerUnavailable:            "local-server-handler-unavailable",
	LocalServerHeaderOnlyEffects:             "local-server-header-only-effects",
	LocalServerIdentityUnavailable:           "local-server-identity-unavailable",
	LocalServerWriterEffectsUnavailable:      "local-server-writer-effects-unavailable",
	NestedInTransferredArgument:              "nested-in-transferred-argument",
	Untouched:                                "none",
	OpaqueConsumption:                        "opaque-consumption",
	OptionalAcquisitionSuccessPhi:            "optional-acquisition-success-phi",
	PairedErrorHelperCleanup:                 "paired-error-helper-cleanup",
	PriorDeferMayCleanCapturedCell:           "prior-defer-may-clean-captured-cell",
	ReleaseProven:                            "release-proven",
	RepeatedGuardEdgeUnknown:                 "repeated-guard-edge-unknown",
	ResourceReturnPath:                       "resource-return-path",
	ReturnedCleanupProjection:                "returned-cleanup-projection",
	ReturnedMayTransfer:                      "returned-may-transfer",
	ReturnedWrapperRetains:                   "returned-wrapper-retains-resource",
	ReturnedProjectionLacksCleanup:           "returned-projection-lacks-cleanup",
	ReturnedViewCannotRelease:                "returned-view-cannot-release",
	RowsTransactionFinished:                  "rows-transaction-finished",
	TransactionContextCanceled:               "transaction-context-canceled",
	ResponseBodyAggregateHandoff:             "response-body-aggregate-handoff",
	SentToChannel:                            "sent-to-channel",
	Settled:                                  "settled",
	StatementParentClosed:                    "statement-parent-closed",
	StoredInMap:                              "stored-in-map",
	IndirectDestinationUnknown:               "indirect-destination-unknown",
	StoredOnCollectionOwner:                  "stored-on-collection-owner",
	WrapperStoredOnForeignOwner:              "wrapper-stored-on-foreign-owner",
	AcquisitionUnreachable:                   "acquisition-unreachable",
	TestifyNoErrorGuard:                      "testify-no-error-guard",
	UnownedReturn:                            "unowned-return",
	UnsummarizedCallee:                       "unsummarized-callee",
	OSIsNotExist:                             "os-isnotexist",
	OSIsExist:                                "os-isexist",
	OSIsPermission:                           "os-ispermission",
	OSIsTimeout:                              "os-istimeout",
	ProcessExitReclaims:                      "process-exit-reclaims",
	ReturnedRetainingWrapper:                 "returned-retaining-wrapper",
}

func (reason Reason) String() string {
	if int(reason) >= len(reasonCodes) {
		return "invalid-resource-reason"
	}
	return reasonCodes[reason]
}
