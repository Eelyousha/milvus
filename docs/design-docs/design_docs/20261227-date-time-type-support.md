# DATE and TIME Type Support (Ponytail Edition)

## Motivation

Add `DATE` and `TIME` scalar types alongside existing `TIMESTAMPTZ`. Users need calendar-date-only (no time, no timezone) and time-of-day-only (no date, no timezone) columns with range query support.

## Key Simplifications (vs TIMESTAMPTZ)

| Component | TIMESTAMPTZ | DATE/TIME (ponytail) |
|-----------|-------------|---------------------|
| Storage | int64 (us from epoch) | DATE: int32 (days), TIME: int64 (us from midnight) |
| Arrow type | int64() | DATE: date32(), TIME: time64(us) |
| Grammar rules | `TimestamptzCompareForward/Reverse` | None — existing `Relational` rule works |
| INTERVAL arithmetic | Yes (TimestamptzArithCompareExpr) | Not needed |
| C++ expression class | `PhyTimestamptzArithCompareExpr` | None — `ExecRangeVisitorImpl<int32_t/int64_t>` works |
| Proto messages | `TimestamptzArithCompareExpr`, `Interval` | None |
| String parsing | `timestamptz` package (~400 lines) | One-liner wrappers (~50 lines) |

## Internal Format

- **DATE**: `int32` days since Unix epoch (1970-01-01). ISO string I/O: `"2024-01-01"`
- **TIME**: `int64` microseconds since midnight. ISO string I/O: `"12:30:00.123456"`

## Range Query Flow

```
User:  date_col < '2024-06-01'
       time_col >= '09:00:00'

Parser (Relational rule):
  → VisitRelational → HandleCompare → handleCompareRightValue → castValue
  → castValue(DATE, "2024-06-01"): parse to int32 days → GenericValue{Int64Val: days}
  → castValue(TIME, "09:00:00"): parse to int64 micros → GenericValue{Int64Val: micros}

C++ ExecRangeVisitorImpl<int32_t>(DATE) / ExecRangeVisitorImpl<int64_t>(TIME):
  → GetValueFromProto<int32_t/int64_t>(GenericValue{kInt64Val}) ✓
  → Existing int32/int64 comparison logic ✓
```

## Implementation

### 1. Proto (milvus-proto/go-api)
- `DataType_Date = 27`, `DataType_Time = 28`
- `DateArray { repeated int32 days = 1; }`
- `TimeArray { repeated int64 microseconds = 1; }`
- `ScalarField` oneof: `DateArray date_data`, `TimeArray time_data`
- `ValueField` oneof: `int64 date_data`, `int64 time_data`

### 2. C++ Types.h
- `GetArrowDataType`: DATE → `arrow::date32()`, TIME → `arrow::time64(us)`
- `ToProtoDataType`: DATE → `proto::schema::Date`, TIME → `proto::schema::Time`
- `TypeTraits<DATE>`, `TypeTraits<TIME>` with NativeType/IsPrimitive/IsFixedWidth
- `fmt::formatter`: `"DATE"`, `"TIME"`
- `MILVUS_DYNAMIC_TYPE_DISPATCH_IMPL`: add DATE, TIME

### 3. C++ Storage
- `storage/Util.cpp`: Arrow builder using `Date32Builder` / `Time64Builder`
- `common/FieldData.cpp`: FillFieldData / FillDefaultValue for DATE/TIME
- `ChunkWriter.cpp`: DATE/TIME write cases
- `segcore/InsertRecord.h`: append_data for int32_t (DATE), int64_t (TIME)

### 4. C++ Expression
- `UnaryExpr.cpp`: DATE → `ExecRangeVisitorImpl<int32_t>`, TIME → `ExecRangeVisitorImpl<int64_t>`
- `TermExpr.cpp`: DATE → `ExecVisitorImpl<int32_t>`, TIME → `ExecVisitorImpl<int64_t>`
- `BinaryRangeExpr.cpp`: same

### 5. Go typeutil
- `IsDateType()`, `IsTimeType()` + size + IsPrimitiveType

### 6. Go datetime parsing (~50 lines)
- `ParseDateISO(s string) (int32, error)` — `time.Parse("2006-01-02")` → days
- `DateDaysToISO(d int32) string` — `time.Unix(int64(d)*86400, 0).UTC().Format(...)`
- `ParseTimeISO(s string) (int64, error)` — `time.Parse("15:04:05.999999")` → microseconds
- `TimeMicrosToISO(us int64) string`

### 7. Go parser
- `castValue`: add DATE/TIME cases — parse string → int64 → `NewInt64(val)`
- `canBeComparedDataType`: DATE/TIME compatible with VarChar (string literal) + JSON
- `castRangeValue`: DATE/TIME pass-through for BinaryRangeExpr

### 8. Go storage layer
- `insert_data.go`, `payload_reader/writer.go`, `field_stats.go`, `data_codec.go`

### 9. Go proxy
- `checkDateFieldData()`, `checkTimeFieldData()` — string → int32/int64 arrays

### 10. Go client
- `ColumnDate` (int32), `ColumnTime` (int64), ISO string variants

### 11. Go Parquet import
- `field_reader.go`: ReadNullableDateData, ReadNullableTimeData

### 12. Tests
- Python: create/insert/range-query with DATE/TIME
- C++: UnaryExpr/BinaryRangeExpr on DATE/TIME