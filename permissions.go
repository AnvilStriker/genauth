package main

type BucketPermissions struct {
	Read  []IAMRole
	Write []IAMRole
}

func (bp *BucketPermissions) GetRoles(operName OperName) (roles []IAMRole) {
	switch operName {
	case "read":
		roles = bp.Read
	case "write":
		roles = bp.Write
	default:
		// complain
	}
	return
}

type QueuePermissions struct {
	Publish   []IAMRole
	Subscribe []IAMRole
}

func (qp *QueuePermissions) GetRoles(operName OperName) (roles []IAMRole) {
	switch operName {
	case "publish":
		roles = qp.Publish
	case "subscribe":
		roles = qp.Subscribe
	default:
		// complain
	}
	return
}

type Permissions struct {
	Buckets             BucketPermissions `json:"buckets"`
	QueuesTopics        QueuePermissions  `json:"queues.topics"`
	QueuesSubscriptions QueuePermissions  `json:"queues.subscriptions"`
}

func (p *Permissions) GetRoles(rsrcKind RsrcKind, operName OperName) (roles []IAMRole) {
	switch rsrcKind {
	case rkBuckets:
		roles = p.Buckets.GetRoles(operName)
	case rkQueuesTopics:
		roles = p.QueuesTopics.GetRoles(operName)
	case rkQueuesSubscriptions:
		roles = p.QueuesSubscriptions.GetRoles(operName)
	default:
		// complain
	}
	return
}
