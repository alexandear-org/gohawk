package resourcelifetime

import (
	"github.com/kojah/gohawk/internal/analysis/analyzers/resources/resourcelifetime/resourcepolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
)

type resourceLifetimeReason = resourcepolicy.Reason

const (
	resourceReasonNone                                     = resourcepolicy.None
	resourceReasonAcquisitionErrorProven                   = resourcepolicy.AcquisitionErrorProven
	resourceReasonAggregateOwnerMayEscape                  = resourcepolicy.AggregateOwnerMayEscape
	resourceReasonAmbiguousCleanupValue                    = resourcepolicy.AmbiguousCleanupValue
	resourceReasonAmbiguousHelperCleanupValue              = resourcepolicy.AmbiguousHelperCleanupValue
	resourceReasonAppended                                 = resourcepolicy.Appended
	resourceReasonBudgetExhausted                          = resourcepolicy.BudgetExhausted
	resourceReasonCallEffectsAsynchronousExposure          = resourcepolicy.CallEffectsAsynchronousExposure
	resourceReasonCallResultMayTransfer                    = resourcepolicy.CallResultMayTransfer
	resourceReasonDirectMayCarry                           = resourcepolicy.DirectMayCarry
	resourceReasonAggregateMayCarry                        = resourcepolicy.AggregateMayCarry
	resourceReasonWrapperMayCarry                          = resourcepolicy.WrapperMayCarry
	resourceReasonParentCleanup                            = resourcepolicy.ParentCleanup
	resourceReasonCapturedAggregateOwner                   = resourcepolicy.CapturedAggregateOwner
	resourceReasonCapturedBodyGuardedCleanup               = resourcepolicy.CapturedBodyGuardedCleanup
	resourceReasonCapturedByLiteralCallingUnreadableCallee = resourcepolicy.CapturedByLiteralCallingUnreadableCallee
	resourceReasonCapturedByPossiblyRetainedCallback       = resourcepolicy.CapturedByPossiblyRetainedCallback
	resourceReasonCapturedByPriorCleanup                   = resourcepolicy.CapturedByPriorCleanup
	resourceReasonCapturedByRetainingLiteral               = resourcepolicy.CapturedByRetainingLiteral
	resourceReasonCapturedByStartedLiteral                 = resourcepolicy.CapturedByStartedLiteral
	resourceReasonCapturedCellMayCleanup                   = resourcepolicy.CapturedCellMayCleanup
	resourceReasonCompressionOutputMayBeAbandoned          = resourcepolicy.CompressionOutputMayBeAbandoned
	resourceReasonCanceledAcquisition                      = resourcepolicy.CanceledAcquisition
	resourceReasonDynamicCallee                            = resourcepolicy.DynamicCallee
	resourceReasonErrorPredicateFalseForNil                = resourcepolicy.ErrorPredicateFalseForNil
	resourceReasonErrorTypeAssertionSucceeded              = resourcepolicy.ErrorTypeAssertionSucceeded
	resourceReasonErrorsAsExactAcquisitionError            = resourcepolicy.ErrorsAsExactAcquisitionError
	resourceReasonErrorsIsNonNilFilesystemSentinel         = resourcepolicy.ErrorsIsNonNilFilesystemSentinel
	resourceReasonErrorsIsNonNilContextSentinel            = resourcepolicy.ErrorsIsNonNilContextSentinel
	resourceReasonEvidenceNotFound                         = resourcepolicy.EvidenceNotFound
	resourceReasonEvidenceUnavailable                      = resourcepolicy.EvidenceUnavailable
	resourceReasonExactErrorEqualsNonNilFilesystemSentinel = resourcepolicy.ExactErrorEqualsNonNilFilesystemSentinel
	resourceReasonHeadAcquisition                          = resourcepolicy.HeadAcquisition
	resourceReasonHeadClientNotUnconfigured                = resourcepolicy.HeadClientNotUnconfigured
	resourceReasonHeadRequestModified                      = resourcepolicy.HeadRequestModified
	resourceReasonHelperCleanupInLoop                      = resourcepolicy.HelperCleanupInLoop
	resourceReasonImportedHelperCleanupInLoop              = resourcepolicy.ImportedHelperCleanupInLoop
	resourceReasonInterfaceMethod                          = resourcepolicy.InterfaceMethod
	resourceReasonHeaderOnlyAcquisition                    = resourcepolicy.HeaderOnlyAcquisition
	resourceReasonLocalServerClientOverrideUnresolved      = resourcepolicy.LocalServerClientOverrideUnresolved
	resourceReasonLocalServerHandlerUnavailable            = resourcepolicy.LocalServerHandlerUnavailable
	resourceReasonLocalServerHeaderOnlyEffects             = resourcepolicy.LocalServerHeaderOnlyEffects
	resourceReasonLocalServerIdentityUnavailable           = resourcepolicy.LocalServerIdentityUnavailable
	resourceReasonLocalServerWriterEffectsUnavailable      = resourcepolicy.LocalServerWriterEffectsUnavailable
	resourceReasonNestedInTransferredArgument              = resourcepolicy.NestedInTransferredArgument
	resourceReasonUntouched                                = resourcepolicy.Untouched
	resourceReasonOpaqueConsumption                        = resourcepolicy.OpaqueConsumption
	optionalAcquisitionSuccessPhi                          = resourcepolicy.OptionalAcquisitionSuccessPhi
	resourceReasonPairedErrorHelperCleanup                 = resourcepolicy.PairedErrorHelperCleanup
	resourceReasonPriorDeferMayCleanCapturedCell           = resourcepolicy.PriorDeferMayCleanCapturedCell
	resourceReasonReleaseProven                            = resourcepolicy.ReleaseProven
	resourceReasonRepeatedGuardEdgeUnknown                 = resourcepolicy.RepeatedGuardEdgeUnknown
	resourceReasonResourceReturnPath                       = resourcepolicy.ResourceReturnPath
	resourceReasonReturnedCleanupProjection                = resourcepolicy.ReturnedCleanupProjection
	resourceReasonReturnedMayTransfer                      = resourcepolicy.ReturnedMayTransfer
	resourceReasonReturnedWrapperRetains                   = resourcepolicy.ReturnedWrapperRetains
	resourceReasonReturnedProjectionLacksCleanup           = resourcepolicy.ReturnedProjectionLacksCleanup
	resourceReasonReturnedViewCannotRelease                = resourcepolicy.ReturnedViewCannotRelease
	resourceReasonRowsTransactionFinished                  = resourcepolicy.RowsTransactionFinished
	resourceReasonTransactionContextCanceled               = resourcepolicy.TransactionContextCanceled
	resourceReasonSentToChannel                            = resourcepolicy.SentToChannel
	resourceReasonResponseBodyAggregateHandoff             = resourcepolicy.ResponseBodyAggregateHandoff
	resourceReasonSettled                                  = resourcepolicy.Settled
	resourceReasonStatementParentClosed                    = resourcepolicy.StatementParentClosed
	resourceReasonStoredInMap                              = resourcepolicy.StoredInMap
	resourceReasonStoredOnCollectionOwner                  = resourcepolicy.StoredOnCollectionOwner
	resourceReasonIndirectDestinationUnknown               = resourcepolicy.IndirectDestinationUnknown
	resourceReasonWrapperStoredOnForeignOwner              = resourcepolicy.WrapperStoredOnForeignOwner
	resourceReasonAcquisitionUnreachable                   = resourcepolicy.AcquisitionUnreachable
	resourceReasonTestifyNoErrorGuard                      = resourcepolicy.TestifyNoErrorGuard
	resourceReasonUnownedReturn                            = resourcepolicy.UnownedReturn
	resourceReasonUnsummarizedCallee                       = resourcepolicy.UnsummarizedCallee
	resourceReasonOSIsNotExist                             = resourcepolicy.OSIsNotExist
	resourceReasonOSIsExist                                = resourcepolicy.OSIsExist
	resourceReasonOSIsPermission                           = resourcepolicy.OSIsPermission
	resourceReasonOSIsTimeout                              = resourcepolicy.OSIsTimeout
	resourceReasonProcessExitReclaims                      = resourcepolicy.ProcessExitReclaims
	resourceReasonReturnedRetainingWrapper                 = resourcepolicy.ReturnedRetainingWrapper
	resourceReasonAppendedToLocalCollection                = resourcepolicy.AppendedToLocalCollection
	resourceReasonCollectionReturned                       = resourcepolicy.CollectionReturned
	resourceReasonCollectionReleasedByHelper               = resourcepolicy.CollectionReleasedByHelper
	resourceReasonCollectionReleased                       = resourcepolicy.CollectionReleased
	resourceReasonCollectionUseUnknown                     = resourcepolicy.CollectionUseUnknown
	resourceReasonRowsExhaustedEdgeUnknown                 = resourcepolicy.RowsExhaustedEdgeUnknown
	resourceReasonResultGuardedDefer                       = resourcepolicy.ResultGuardedDefer
	resourceReasonResultGuardedRelease                     = resourcepolicy.ResultGuardedRelease
	resourceReasonResultGuardedUnknown                     = resourcepolicy.ResultGuardedUnknown
	resourceReasonMemoryWriter                             = resourcepolicy.MemoryWriter
	resourceReasonAcquisitionLocationUnknown               = resourcepolicy.AcquisitionLocationUnknown
	resourceReasonPriorDeferMayRelease                     = resourcepolicy.PriorDeferMayRelease
	resourceReasonCount                                    = resourcepolicy.Count
)

// resourceProof records analyzer-owned evidence. It does not inject resource
// API contracts into the shared SSA reason domain.
type resourceProof resourcepolicy.Evidence

func (proof resourceProof) Proven() bool { return proof.State == proofs.EvidenceProven }
