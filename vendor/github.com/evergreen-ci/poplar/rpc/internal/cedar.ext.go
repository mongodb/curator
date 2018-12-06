package internal

import (
	"github.com/evergreen-ci/poplar"
	"github.com/golang/protobuf/ptypes"
)

func ExportArtifactInfo(in *poplar.TestArtifact) *ArtifactInfo {
	out := &ArtifactInfo{
		Bucket: in.Bucket,
		Path:   in.Path,
		Tags:   in.Tags,
	}

	if ts, err := ptypes.TimestampProto(in.CreatedAt); err == nil {
		out.CreatedAt = ts
	}

	switch {
	case in.PayloadFTDC:
		out.Format = DataFormat_FTDC
	case in.PayloadBSON:
		out.Format = DataFormat_BSON
	}

	switch {
	case in.DataUncompressed:
		out.Compression = CompressionType_NONE
	case in.DataGzipped:
		out.Compression = CompressionType_GZ
	case in.DataTarball:
		out.Compression = CompressionType_TARGZ
	}

	switch {
	case in.EventsRaw:
		out.Schema = SchemaType_RAW_EVENTS
	case in.EventsHistogram:
		out.Schema = SchemaType_HISTOGRAM
	case in.EventsIntervalSummary:
		out.Schema = SchemaType_INTERVAL_SUMMARIZATION
	case in.EventsCollapsed:
		out.Schema = SchemaType_COLLAPSED_EVENTS
	}

	return out
}

func ExportRollup(in *poplar.TestMetrics) *RollupValue {
	out := &RollupValue{
		Name:          in.Name,
		Version:       int64(in.Version),
		UserSubmitted: true,
	}

	switch val := in.Value.(type) {
	case int64:
		out.Value = &RollupValue_Int{Int: val}
	case float64:
		out.Value = &RollupValue_Fl{Fl: val}
	case int:
		out.Value = &RollupValue_Int{Int: int64(val)}
	case int32:
		out.Value = &RollupValue_Int{Int: int64(val)}
	case float32:
		out.Value = &RollupValue_Fl{Fl: float64(val)}
	}

	return out
}
