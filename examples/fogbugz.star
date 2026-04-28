# Example: FogBugz API credential injection.
#
# The fogbugz-mcp server POSTs JSON with { "token": "...", ...params }
# to /api/<command>. This script replaces the token in the body with
# the real one from the macOS Keychain.
#
# Usage:
#   mcpsafe -config examples/fogbugz.star -backend https://hmssupport.fogbugz.com
#
# Prerequisites:
#   security add-generic-password -s fogbugz-token -a fogbugz-token -w "$(pbpaste)"

def transform(req):
    if "body" in req:
        req["body"]["token"] = keychain("fogbugz-token")
    return req
