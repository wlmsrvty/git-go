- [ ] git internals: https://git-scm.com/docs/githooks
- [ ] structurize go project: https://go.dev/doc/modules/layout
- [ ] staging area: git write-tree assumes that all the files in the working directory are in the staging area


https://www.git-scm.com/docs/http-protocol

- [ ] status
- [ ] log
- [ ] diff
- [ ] add
- [x] commit
    - [ ] Staging area
    - [x] Commit object
- [ ] clone
    - [ ] Smart HTTP Protocol
    - [ ] Dumb  HTTP Protocol
    - [ ] Git Protocol
    - [ ] SSH Protocol
    - [ ] no sideband capability used
    - [ ] deltas unpacking


Full clone implementation:
- Reference discovery
- Packfile negotiation
- Packfile transfer and unpacking

Clone implementation:
- using smart HTTP protocol only
- using loose objects: unpacking packfile fully to .git/objects
- not using packfile index .idx

## TODO
- add
- commit
- log
- status