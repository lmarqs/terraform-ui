resource "terraform_data" "keyed" {
  for_each = toset(["x y", "a,b", "plain"])
  input    = each.key
}
