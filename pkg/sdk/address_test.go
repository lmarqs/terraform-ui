package sdk

import "testing"

func TestAddress_String_WhenCalled_ShouldReturnUnderlyingValue(t *testing.T) {
	tests := []struct {
		name string
		addr Address
		want string
	}{
		{"ShouldReturnSimpleResource", Address("aws_instance.main"), "aws_instance.main"},
		{"ShouldReturnModuleResource", Address("module.vpc.aws_subnet.private[0]"), "module.vpc.aws_subnet.private[0]"},
		{"ShouldReturnEmptyForZeroValue", Address(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.String()
			if got != tt.want {
				t.Errorf("Address(%q).String() = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestAddress_WhenConstructedFromLiteral_ShouldPreserveValue(t *testing.T) {
	tests := []struct {
		name    string
		literal string
	}{
		{"ShouldPreserveResourceAddress", "aws_s3_bucket.logs"},
		{"ShouldPreserveDataSource", "data.aws_ami.ubuntu"},
		{"ShouldPreserveIndexedResource", "aws_instance.web[2]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := Address(tt.literal)
			if string(addr) != tt.literal {
				t.Errorf("Address(%q) underlying value = %q, want %q", tt.literal, string(addr), tt.literal)
			}
		})
	}
}

func TestAddress_WhenZeroValue_ShouldBeEmpty(t *testing.T) {
	var addr Address
	if addr.String() != "" {
		t.Errorf("zero value Address.String() = %q, want %q", addr.String(), "")
	}
}
