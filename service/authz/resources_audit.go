package authz

const ResourceAudit = "audit"

var AuditRead = Permission{Resource: ResourceAudit, Action: ActionRead}
