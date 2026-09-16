## ADDED Requirements

### Requirement: HR grouped advertisement count
The system SHALL show «Количество объявлений» immediately left of «Показы» for employee and object groups of reports explicitly uploaded as HR.

#### Scenario: Multiple advertisements in a group
- **WHEN** an HR report has two parsed rows for one employee/object, including a zero-activity row
- **THEN** that group has count 2, independent of IDs or shows.

#### Scenario: Other report types and groupings
- **WHEN** an Avito report or city/category/name/offers grouping is displayed
- **THEN** existing columns remain unchanged.

### Requirement: Count across result formats
The system SHALL carry group counts into ordinary results, comparison periods, and XLSX HR sections.

#### Scenario: Period comparison
- **WHEN** two HR periods are compared by employee/object
- **THEN** count P1, P2 and Δ=P2−P1 precede shows; missing groups have zero counts.

#### Scenario: XLSX download
- **WHEN** selected reports are exported
- **THEN** only HR employee/object sections gain the numeric count column and other metrics retain their alignment.
