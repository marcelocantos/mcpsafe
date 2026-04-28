// Copyright 2026 Marcelo Cantos
// SPDX-License-Identifier: Apache-2.0

package keychain

/*
#cgo LDFLAGS: -framework CoreFoundation -framework Security
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>

typedef struct {
	char *data;
	int   len;
	int   status;
} KeychainResult;

KeychainResult keychain_find(const char *service, const char *account) {
	KeychainResult result = {NULL, 0, 0};

	CFStringRef cfService = CFStringCreateWithCString(NULL, service, kCFStringEncodingUTF8);
	CFStringRef cfAccount = CFStringCreateWithCString(NULL, account, kCFStringEncodingUTF8);

	const void *keys[] = {
		kSecClass,
		kSecAttrService,
		kSecAttrAccount,
		kSecReturnData,
		kSecMatchLimit,
	};
	const void *values[] = {
		kSecClassGenericPassword,
		cfService,
		cfAccount,
		kCFBooleanTrue,
		kSecMatchLimitOne,
	};

	CFDictionaryRef query = CFDictionaryCreate(
		NULL, keys, values, 5,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks);

	CFDataRef data = NULL;
	OSStatus status = SecItemCopyMatching(query, (CFTypeRef *)&data);

	CFRelease(query);
	CFRelease(cfService);
	CFRelease(cfAccount);

	result.status = (int)status;
	if (status == errSecSuccess && data != NULL) {
		CFIndex len = CFDataGetLength(data);
		result.data = (char *)malloc(len);
		CFDataGetBytes(data, CFRangeMake(0, len), (UInt8 *)result.data);
		result.len = (int)len;
		CFRelease(data);
	}

	return result;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func readKeychain(service, account string) (string, error) {
	cService := C.CString(service)
	defer C.free(unsafe.Pointer(cService))
	cAccount := C.CString(account)
	defer C.free(unsafe.Pointer(cAccount))

	result := C.keychain_find(cService, cAccount)

	if result.status != 0 {
		return "", fmt.Errorf("keychain lookup failed for service=%q account=%q: OSStatus %d", service, account, result.status)
	}

	if result.data == nil {
		return "", fmt.Errorf("keychain lookup returned no data for service=%q account=%q", service, account)
	}

	pw := C.GoStringN(result.data, result.len)
	C.free(unsafe.Pointer(result.data))
	return pw, nil
}
