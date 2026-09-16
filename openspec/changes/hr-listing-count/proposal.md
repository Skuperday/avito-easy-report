# HR advertisement counts

## Why
HR employee/object reports need the number of advertisements beside impressions.

## What Changes
- Count parsed offer rows per non-empty employee/object group.
- Preserve the cabinet upload's explicit `type=hr` on stored reports and expose report type to results.
- Show «Количество объявлений» immediately before «Показы» only for HR employee/object tables, including comparison P1/P2/Δ and XLSX.
- Leave other group columns and Avito reports unchanged.

## Impact
Go aggregation/DTOs/upload/export; Nuxt results; regression tests. Local only.
