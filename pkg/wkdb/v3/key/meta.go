package key

func EncodeSystemUIDRowKey(uid string) []byte {
	key := newMetaKey(ScopeMetaPrimary, TableSystemUID, KindRow)
	return appendString(key, uid)
}

func ParseSystemUIDRowKey(key []byte) (uid string, err error) {
	off, err := parseMetaKeyHeader(key, ScopeMetaPrimary, TableSystemUID, KindRow)
	if err != nil {
		return "", err
	}
	uid, _, err = readString(key, off)
	return
}

func EncodeTesterRowKey(no string) []byte {
	key := newMetaKey(ScopeMetaPrimary, TableTester, KindRow)
	return appendString(key, no)
}

func ParseTesterRowKey(key []byte) (no string, err error) {
	off, err := parseMetaKeyHeader(key, ScopeMetaPrimary, TableTester, KindRow)
	if err != nil {
		return "", err
	}
	no, _, err = readString(key, off)
	return
}

func EncodePluginRowKey(no string) []byte {
	key := newMetaKey(ScopeMetaPrimary, TablePlugin, KindRow)
	return appendString(key, no)
}

func ParsePluginRowKey(key []byte) (no string, err error) {
	off, err := parseMetaKeyHeader(key, ScopeMetaPrimary, TablePlugin, KindRow)
	if err != nil {
		return "", err
	}
	no, _, err = readString(key, off)
	return
}

func EncodePluginUserRowKey(pluginNo, uid string) []byte {
	key := newMetaKey(ScopeMetaPrimary, TablePluginUser, KindRow)
	key = appendString(key, pluginNo)
	return appendString(key, uid)
}

func ParsePluginUserRowKey(key []byte) (pluginNo, uid string, err error) {
	off, err := parseMetaKeyHeader(key, ScopeMetaPrimary, TablePluginUser, KindRow)
	if err != nil {
		return "", "", err
	}
	pluginNo, off, err = readString(key, off)
	if err != nil {
		return "", "", err
	}
	uid, _, err = readString(key, off)
	return
}

func PluginUserPluginRange(pluginNo string) (lower, upper []byte) {
	lower = newMetaKey(ScopeMetaPrimary, TablePluginUser, KindRow)
	lower = appendString(lower, pluginNo)
	return lower, PrefixUpperBound(lower)
}

func EncodePluginUserUIDSecondIndexKey(uid, pluginNo string) []byte {
	key := newMetaKey(ScopeMetaSecondIdx, TablePluginUser, KindSecondIndex)
	key = appendString(key, uid)
	return appendString(key, pluginNo)
}

func ParsePluginUserUIDSecondIndexKey(key []byte) (uid, pluginNo string, err error) {
	off, err := parseMetaKeyHeader(key, ScopeMetaSecondIdx, TablePluginUser, KindSecondIndex)
	if err != nil {
		return "", "", err
	}
	uid, off, err = readString(key, off)
	if err != nil {
		return "", "", err
	}
	pluginNo, _, err = readString(key, off)
	return
}

func PluginUserUIDSecondIndexRange(uid string) (lower, upper []byte) {
	lower = newMetaKey(ScopeMetaSecondIdx, TablePluginUser, KindSecondIndex)
	lower = appendString(lower, uid)
	return lower, PrefixUpperBound(lower)
}
