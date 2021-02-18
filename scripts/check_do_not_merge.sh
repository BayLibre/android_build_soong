#!/bin/bash
if git show -s --format=%s $1 | grep -qE '(DO NOT MERGE)|(RESTRICT AUTOMERGE)'; then
    echo 'DO NOT MERGE and RESTRICT AUTOMERGE very often lead to unintended results' >&2
    echo 'and are not allowed to be used in this project.' >&2
    echo 'Please use the Merged-In tag to be more explicit about where this change' >&2
    echo 'should merge to. Google-internal documentation exists at go/merged-in'
    echo >&2
    echo 'If this check is mis-triggering or you know Merged-In is incorrect in this' >&2
    echo 'situation you can bypass this check with `repo upload --no-verify`.' >&2
    exit 1
fi
