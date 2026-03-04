package mediainfo

// NormaliseCodec is exported for testing only.
func NormaliseCodec(name string) string {
	return normaliseCodec(name)
}

// NormaliseResolution is exported for testing only.
func NormaliseResolution(height int) string {
	return normaliseResolution(height)
}

// DetectHDRTest is exported for testing only. It constructs a minimal stream
// and calls detectHDR so callers don't have to import ffprobe internals.
func DetectHDRTest(colorTransfer, sideDataType string) string {
	var sd []ffprobeSideData
	if sideDataType != "" {
		sd = append(sd, ffprobeSideData{SideDataType: sideDataType})
	}
	st := &ffprobeStream{
		ColorTransfer: colorTransfer,
		SideDataList:  sd,
	}
	return detectHDR(st)
}
