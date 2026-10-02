# Design

The cabinet has no persisted type field. Its report-type selector sends multipart `type`, currently ignored by UploadReport. Preserve it per report (`hr`, default `avito`) rather than infer HR from group names or a mutable selector. Old in-memory reports without a type remain non-HR; no database migration.

`listingCount` counts parsed rows (the existing upload `rows` contract), not unique listing IDs: missing/repeated IDs count, zero-activity rows count. Empty grouping keys remain excluded. Aggregate counts travel through Stats → ResultStats. UI/XLSX gates on reportType plus employee/object grouping. Comparison counts apply only when both selected endpoint periods are HR; disappearing HR groups need zero late values and negative deltas. Comparison export continues to export selected reports separately, matching the existing download contract.

Testing: Go behavioral aggregation, HTTP upload/type propagation, XLSX readback; render the actual Vue results SFC with deterministic API fixtures, click grouping/sort controls, and inspect DOM. No production credentials/data required.
