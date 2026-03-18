package parser

import (
	"regexp"
	"strings"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// All regexps are compiled once at package init — never inside parse functions.

var (
	// ── Resolution ───────────────────────────────────────────────────────
	re2160p = regexp.MustCompile(`(?i)(?:4k|uhd|2160p)`)
	re1080p = regexp.MustCompile(`(?i)1080p`)
	re720p  = regexp.MustCompile(`(?i)720p`)
	re576p  = regexp.MustCompile(`(?i)576p`)
	re480p  = regexp.MustCompile(`(?i)480p`)

	// ── Source ───────────────────────────────────────────────────────────
	reRawHD     = regexp.MustCompile(`(?i)\braw[\s\-]?hd\b`)
	reBRDisk    = regexp.MustCompile(`(?i)(?:\bbdmv\b|\bbd25\b|\bbd50\b|\bbr[\s\-]?disk\b)`)
	reRemux     = regexp.MustCompile(`(?i)(?:remux|bdremux)`)
	reBluRay    = regexp.MustCompile(`(?i)blu[\s\-]?ray|bluray`)
	reWEBDL     = regexp.MustCompile(`(?i)web[\s\-.]?dl`)
	reWEBRip    = regexp.MustCompile(`(?i)web[\s\-.]?rip`)
	reHDTV      = regexp.MustCompile(`(?i)hdtv`)
	reDVDSCR    = regexp.MustCompile(`(?i)(?:\bdvdscr\b|\bscreener\b|\bscr\b)`)
	reDVDR      = regexp.MustCompile(`(?i)(?:\bdvd[\s\-]?r\b|\bdvd9\b|\bdvd5\b|\bdvdr\b)`)
	reDVDRip    = regexp.MustCompile(`(?i)dvd[\s\-.]?rip`)
	reRegional  = regexp.MustCompile(`(?i)(?:\br5\b|\bregional\b)`)
	reTelecine  = regexp.MustCompile(`(?i)(?:\btelecine\b|\bhdtc\b|\btc\b)`)
	reTelesync  = regexp.MustCompile(`(?i)(?:\btelesync\b|\bhdts\b|\bpdvd\b|\bts\b)`)
	reCAM       = regexp.MustCompile(`(?i)(?:hd)?cam(?:rip)?`)
	reWorkprint = regexp.MustCompile(`(?i)(?:\bworkprint\b|\bwp\b)`)

	// ── HDR ──────────────────────────────────────────────────────────────
	reDolbyVision = regexp.MustCompile(`(?i)(?:\bdv\b|dovi|dolby[\s\-.]?vision)`)
	reHDR10Plus   = regexp.MustCompile(`(?i)(?:hdr10\+|hdr10plus)`)
	reHDR10       = regexp.MustCompile(`(?i)hdr(?:10)?`)
	reHLG         = regexp.MustCompile(`(?i)\bhlg\b`)

	// ── Codec ────────────────────────────────────────────────────────────
	reX265 = regexp.MustCompile(`(?i)x265|h[\s\-.]?265|hevc`)
	reX264 = regexp.MustCompile(`(?i)x264|h[\s\-.]?264|avc`)
	reAV1  = regexp.MustCompile(`(?i)\bav1\b`)
	reXVID = regexp.MustCompile(`(?i)xvid|divx`)
)

func parseSource(norm string) plugin.Source {
	switch {
	case reRawHD.MatchString(norm):
		return plugin.SourceRawHD
	case reBRDisk.MatchString(norm):
		return plugin.SourceBRDisk
	case reRemux.MatchString(norm):
		return plugin.SourceRemux
	case reBluRay.MatchString(norm):
		return plugin.SourceBluRay
	case reWEBDL.MatchString(norm):
		return plugin.SourceWEBDL
	case reWEBRip.MatchString(norm):
		return plugin.SourceWEBRip
	case reHDTV.MatchString(norm):
		return plugin.SourceHDTV
	case reDVDSCR.MatchString(norm):
		return plugin.SourceDVDSCR
	case reDVDR.MatchString(norm):
		return plugin.SourceDVDR
	case reDVDRip.MatchString(norm):
		return plugin.SourceDVD
	case reRegional.MatchString(norm):
		return plugin.SourceRegional
	case reTelecine.MatchString(norm):
		return plugin.SourceTELECINE
	case reTelesync.MatchString(norm):
		return plugin.SourceTelesync
	case reCAM.MatchString(norm):
		return plugin.SourceCAM
	case reWorkprint.MatchString(norm):
		return plugin.SourceWorkprint
	default:
		return plugin.SourceUnknown
	}
}

func parseResolution(norm string, src plugin.Source) plugin.Resolution {
	switch {
	case re2160p.MatchString(norm):
		return plugin.Resolution2160p
	case re1080p.MatchString(norm):
		return plugin.Resolution1080p
	case re720p.MatchString(norm):
		return plugin.Resolution720p
	case re576p.MatchString(norm):
		return plugin.Resolution576p
	case re480p.MatchString(norm):
		return plugin.Resolution480p
	case src == plugin.SourceDVD || src == plugin.SourceDVDR || src == plugin.SourceDVDSCR || src == plugin.SourceRegional:
		return plugin.ResolutionSD
	default:
		return plugin.ResolutionUnknown
	}
}

func parseHDR(norm string) plugin.HDRFormat {
	switch {
	case reDolbyVision.MatchString(norm):
		return plugin.HDRDolbyVision
	case reHDR10Plus.MatchString(norm):
		return plugin.HDRHDR10Plus
	case reHDR10.MatchString(norm):
		return plugin.HDRHDR10
	case reHLG.MatchString(norm):
		return plugin.HDRHLG
	default:
		return plugin.HDRNone
	}
}

func parseCodec(norm string) plugin.Codec {
	switch {
	case reX265.MatchString(norm):
		return plugin.CodecX265
	case reX264.MatchString(norm):
		return plugin.CodecX264
	case reAV1.MatchString(norm):
		return plugin.CodecAV1
	case reXVID.MatchString(norm):
		return plugin.CodecXVID
	default:
		return plugin.CodecUnknown
	}
}

// buildQualityName produces the human-readable quality label.
func buildQualityName(res plugin.Resolution, src plugin.Source, codec plugin.Codec, hdr plugin.HDRFormat) string {
	srcLabel := sourceLabel(src)
	resLabel := resolutionLabel(res)
	codecLabel := codecLabel(codec)
	hdrLabel := hdrLabel(hdr)

	var sb strings.Builder
	if srcLabel != "" && resLabel != "" {
		sb.WriteString(srcLabel)
		sb.WriteByte('-')
		sb.WriteString(resLabel)
	} else if srcLabel != "" {
		sb.WriteString(srcLabel)
	} else if resLabel != "" {
		sb.WriteString(resLabel)
	}
	if codecLabel != "" {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(codecLabel)
	}
	if hdrLabel != "" {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(hdrLabel)
	}
	return sb.String()
}

func sourceLabel(src plugin.Source) string {
	switch src {
	case plugin.SourceRawHD:
		return "Raw-HD"
	case plugin.SourceBRDisk:
		return "BR-DISK"
	case plugin.SourceRemux:
		return "Bluray Remux"
	case plugin.SourceBluRay:
		return "Bluray"
	case plugin.SourceWEBDL:
		return "WEBDL"
	case plugin.SourceWEBRip:
		return "WEBRip"
	case plugin.SourceHDTV:
		return "HDTV"
	case plugin.SourceDVDSCR:
		return "DVDSCR"
	case plugin.SourceDVDR:
		return "DVD-R"
	case plugin.SourceDVD:
		return "DVD"
	case plugin.SourceRegional:
		return "Regional"
	case plugin.SourceTELECINE:
		return "Telecine"
	case plugin.SourceTelesync:
		return "Telesync"
	case plugin.SourceCAM:
		return "CAM"
	case plugin.SourceWorkprint:
		return "Workprint"
	default:
		return ""
	}
}

func resolutionLabel(res plugin.Resolution) string {
	switch res {
	case plugin.Resolution2160p:
		return "2160p"
	case plugin.Resolution1080p:
		return "1080p"
	case plugin.Resolution720p:
		return "720p"
	case plugin.Resolution576p:
		return "576p"
	case plugin.Resolution480p:
		return "480p"
	case plugin.ResolutionSD:
		return "SD"
	default:
		return ""
	}
}

func codecLabel(codec plugin.Codec) string {
	switch codec {
	case plugin.CodecX265:
		return "x265"
	case plugin.CodecX264:
		return "x264"
	case plugin.CodecAV1:
		return "AV1"
	case plugin.CodecXVID:
		return "XviD"
	default:
		return ""
	}
}

func hdrLabel(hdr plugin.HDRFormat) string {
	switch hdr {
	case plugin.HDRHDR10:
		return "HDR10"
	case plugin.HDRDolbyVision:
		return "Dolby Vision"
	case plugin.HDRHLG:
		return "HLG"
	case plugin.HDRHDR10Plus:
		return "HDR10+"
	default:
		return ""
	}
}
