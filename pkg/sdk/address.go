package sdk

// Address is a terraform resource address (e.g., "aws_instance.web",
// "module.vpc.aws_subnet.private[0]"). No validation — terraform owns format.
type Address string

func (a Address) String() string { return string(a) }
