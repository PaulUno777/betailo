package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AdminAction is one line of the back office's journal: who did what
// to which target. Written for every staff action that changes
// something, and for every look at an identity document.
type AdminAction struct {
	ID bson.ObjectID
	// ActorID is the staff member; ActorName and ActorRole are frozen
	// at the time of the action, so the journal still reads right after
	// a role change.
	ActorID   bson.ObjectID
	ActorName string
	ActorRole StaffRole
	// Action is a stable code ("farm.approve"); clients translate it.
	Action string
	// TargetType names the collection TargetID points into ("farm",
	// "user", "reservation", "report"); TargetLabel is what a person
	// would call it (a farm's name, an account's email, "RSV-0042").
	TargetType  string
	TargetID    bson.ObjectID
	TargetLabel string
	// Detail is the free text that went with the action: a refusal
	// reason, the new farm limit.
	Detail    string
	CreatedAt time.Time
}

// Journal action codes.
const (
	ActionFarmApprove     = "farm.approve"
	ActionFarmReject      = "farm.reject"
	ActionFarmSuspend     = "farm.suspend"
	ActionFarmProofViewed = "farm.proof_viewed"
	ActionUserSuspend     = "user.suspend"
	ActionUserReactivate  = "user.reactivate"
	ActionFarmLimitSet    = "user.farm_limit"
	ActionEscrowRelease   = "escrow.release"
	ActionEscrowRefund    = "escrow.refund"
	ActionReportResolve   = "report.resolve"
	ActionReportNote      = "report.note"
	ActionEscrowExport    = "escrow.export"
	ActionStaffGrant      = "staff.grant"
	ActionStaffRevoke     = "staff.revoke"
	ActionStaffInvite     = "staff.invite"
	// ActionStaffInviteCancel withdraws a pending invitation.
	ActionStaffInviteCancel = "staff.invite_cancel"
	// ActionCatalogPublish publishes a new version of the reference data.
	ActionCatalogPublish = "catalog.publish"
)

// MaxAuditDetailLen bounds AdminAction.Detail.
const MaxAuditDetailLen = 600
