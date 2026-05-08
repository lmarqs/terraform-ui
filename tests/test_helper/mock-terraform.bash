#!/usr/bin/env bash

_mock_terraform_setup() {
  MOCK_DIR="$BATS_TEST_TMPDIR/mock-bin"
  mkdir -p "$MOCK_DIR"

  cat > "$MOCK_DIR/terraform" <<'MOCK'
#!/usr/bin/env bash
case "$1" in
  plan)
    echo "module.a.resource_b: Refreshing state..."
    echo "module.a.resource_a: Refreshing state..."
    echo "module.b.resource_c: Reading..."
    for arg in "$@"; do
      if [[ "$arg" == -out=* ]]; then
        touch "${arg#-out=}"
      fi
    done
    ;;
  show)
    cat <<'JSON'
{"resource_changes":[
  {"address":"module.a.resource_b","change":{"actions":["create"]}},
  {"address":"module.a.resource_a","change":{"actions":["update"]}},
  {"address":"module.b.resource_c","change":{"actions":["delete"]}}
]}
JSON
    ;;
  apply)
    echo "module.a.resource_b: Creating..."
    echo "module.a.resource_b: Creation complete after 1s"
    echo "module.a.resource_a: Modifying..."
    echo "module.a.resource_a: Modifications complete after 1s"
    ;;
  state)
    echo "module.a.resource_a"
    echo "module.a.resource_b"
    echo "module.b.resource_c"
    ;;
esac
MOCK
  chmod +x "$MOCK_DIR/terraform"
  PATH="$MOCK_DIR:$PATH"
}
