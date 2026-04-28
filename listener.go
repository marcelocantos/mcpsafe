// Copyright 2026 Marcelo Cantos
// SPDX-License-Identifier: Apache-2.0

package main

import "net"

func newListener(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
