terraform {
  bin = "terraform"
}

chdir {
  members = ["modules/bad"]
}
