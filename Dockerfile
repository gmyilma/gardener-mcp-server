# SPDX-FileCopyrightText: 2026 Girma Yilma
#
# SPDX-License-Identifier: Apache-2.0

# Runtime image for service mode.
#
# The binary is built by goreleaser and copied in, rather than compiled here, so
# that the released archive and the image contain the identical artifact.
#
# distroless/static holds no shell, no package manager and no libc. If the
# server is ever compromised (the "MCP server compromise" row of the threat
# model, review section 4), there is nothing in the image to pivot with, and the
# blast radius stays limited to the short-lived read-only credentials it holds.
FROM gcr.io/distroless/static-debian12:nonroot

# Non-root, matching the distroless nonroot user. The Helm chart should also set
# readOnlyRootFilesystem, drop ALL capabilities, and disallow privilege
# escalation; nothing here needs to write to disk.
USER 65532:65532

COPY gardener-mcp-server /usr/local/bin/gardener-mcp-server

# Service mode listens over streamable HTTP. Local mode uses stdio and ignores
# this entirely.
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/gardener-mcp-server"]
CMD ["--mode=service"]
