wait ready
key P
wait ready
assert view Detected 2 phantom change(s) out of 3 total
assert view aws_security_group.web
assert view aws_iam_role_policy.deploy
key enter
wait ready
assert view Reason: tag/label ordering difference (cosmetic)
key q
