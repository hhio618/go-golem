package props

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	NodeInfoName      = "Name"
	NodeInfoSubnetTag = "SubnetTag"
)

// NodeInfo holds the properties describing the information regarding the node.
type NodeInfo struct {
	Model
	// Name is the human-readable name of the Golem node.
	Name string `field:"optional"`
	// SubnetTag is the the name of the subnet within which the Demands and Offers are matched.
	SubnetTag string `field:"optional"`
}

func (ni *NodeInfo) Keys() map[string]string {
	return map[string]string{
		NodeInfoName:      "golem.node.id.name",
		NodeInfoSubnetTag: "golem.node.debug.subnet",
	}
}

var NodeInfoKeys = (&NodeInfo{}).Keys()

const (
	ActivityCostCap       = "CostCap"
	ActivityCostWarning   = "CostWarning"
	ActivityTimeoutSecs   = "TimeoutSecs"
	ActivityExpiration    = "Expiration"
	ActivityMultiActivity = "MultiActivity"
)

// Activity-related Properties.
type Activity struct {
	Model
	/* CostCap sets a Hard cap on total cost of the Activity (regardless of the usage vector or
	pricing function). The Provider is entitled to 'kill' an Activity which exceeds the
	capped cost amount indicated by Requestor.
	*/
	CostCap decimal.Decimal `field:"optional"`
	/*CostWarning sets a Soft cap on total cost of the Activity (regardless of the usage vector or
	pricing function). When the cost_warning amount is reached for the Activity,
	the Provider is expected to send a Debit Note to the Requestor, indicating
	the current amount due
	*/
	CostWarning decimal.Decimal `field:"optional"`
	/* TimeoutSecs is a timeout value for batch computation (eg. used for container-based batch
	processes). This property allows to set the timeout to be applied by the Provider
	when running a batch computation: the Requestor expects the Activity to take
	no longer than the specified timeout value - which implies that
	eg. the golem.usage.duration_sec counter shall not exceed the specified
	timeout value.
	*/
	TimeoutSecs float32   `field:"optional"`
	Expiration  time.Time `field:"optional"`
	// MultiActivity means whether client supports multi_activity (executing more than one activity per agreement).
	MultiActivity bool `field:"optional"`
}

func (a *Activity) Keys() map[string]string {
	return map[string]string{

		ActivityCostCap:       "golem.activity.cost_cap",
		ActivityCostWarning:   "golem.activity.cost_warning",
		ActivityTimeoutSecs:   "golem.activity.timeout_secs",
		ActivityExpiration:    "golem.srv.comp.expiration",
		ActivityMultiActivity: "golem.srv.caps.multi-activity",
	}
}

var ActivityKeys = (&Activity{}).Keys()
