# direktiv-actions

This action synchronises workflow folders or single workflow iwth direktiv.

Because this action needs the history of workflows it needs to checkout the repository
where it is being used with `fetch-depth: 0`, e.g.:

```
steps:
  - uses: actions/checkout@v2
    with:
      fetch-depth: 0
```

## Usage

See [action.yaml](action.yaml)

### Basic

```yaml
- name: execute
  id: execute
  with:
    server: my-direktiv-server
    namespace: my-namespace
    sync: tests/wf.yaml
  uses: vorteil/direktiv-actions-ghsync@test
```


### Folder

```yaml
- name: execute
  id: execute
  with:
    server: my-direktiv-server
    names: project/workflows
  uses: vorteil/direktiv-actions-ghsync@test
```

### Using authentication token

```yaml
- name: execute
  id: execute
  with:
    server: my-direktiv-server
    namespace: my-namespace
    sync: tests/wf.yaml
    token: ${{ secrets.DIREKTIV_TOKEN }}
  uses: vorteil/direktiv-actions-ghsync@test
```
