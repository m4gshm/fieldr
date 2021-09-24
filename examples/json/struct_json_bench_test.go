package json

import (
	"encoding/json"
	"strings"
	"testing"
)

var benchStruct = Struct{
	ID:     1,
	Name:   "Name",
	NoJson: "NoJson",
}

func Benchmark_TestStruct_MarshalJSON(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := benchStruct.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()
}

func Benchmark_TestStruct_MarshalJSON_SharedBuilder(b *testing.B) {
	b.ResetTimer()

	var builder strings.Builder

	for i := 0; i < b.N; i++ {
		err := benchStruct.MarshalJSONToBuilder(&builder)
		_ = builder.String()
		if err != nil {
			b.Fatal(err)
		}
	}

	builder.Reset()

	b.StopTimer()
}

func Benchmark_TestStruct_MarshalJSON_SharedBuilder_32(b *testing.B) {
	b.ResetTimer()

	mashalToBuilder(b, 32)

	b.StopTimer()
}

func Benchmark_TestStruct_MarshalJSON_SharedBuilder_64(b *testing.B) {
	b.ResetTimer()

	mashalToBuilder(b, 64)

	b.StopTimer()
}

func Benchmark_TestStruct_MarshalJSON_SharedBuilder_128(b *testing.B) {
	b.ResetTimer()

	mashalToBuilder(b, 128)

	b.StopTimer()
}

func Benchmark_TestStruct_MarshalJSON_SharedBuilder_256(b *testing.B) {
	b.ResetTimer()

	mashalToBuilder(b, 256)

	b.StopTimer()
}

func mashalToBuilder(b *testing.B, n int) {
	var builder strings.Builder
	builder.Grow(n)

	for i := 0; i < b.N; i++ {
		err := benchStruct.MarshalJSONToBuilder(&builder)
		_ = builder.String()
		if err != nil {
			b.Fatal(err)
		}
	}

	builder.Reset()
	builder.Grow(n)
}

func Benchmark_TestStruct_DefaultMarshalJSON(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(benchStruct)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()
}
