# Ralph Action List — P1-03 Real CRUD Scaffolds

## Goal

Deliver the first genuinely useful starter CRUD scaffold path.

## Action stack

### Action 1 — failing proof first
- add failing starter/generated-app CRUD proof
- choose one canonical starter CRUD target resource
- prove the current generator surface is still too thin

### Action 2 — first useful scaffold output
- upgrade the starter-safe generator path to emit real CRUD artifacts
- include list/show/create/update/delete route/page/test surfaces

### Action 3 — validation integration
- use the P1-02 validation seam on create/update paths
- prove invalid form submissions fail usefully

### Action 4 — destroy/idempotency safety
- verify generated output remains reversible
- keep `destroy resource:<name>` safe and truthful

### Action 5 — product-truth decision
- decide whether starter `make:scaffold` should remain closed or reopen narrowly
- only reopen if proof clearly supports it

## Required evidence before completion
- generated-app CRUD proof green
- starter smoke/build proof green
- destroy safety green
- docs/help aligned if support surface changed
