CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    owner STRING NOT NULL,
    url STRING NOT NULL,
    notification_index INT4 DEFAULT 0,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL,
    cells INT64[] NOT NULL,
    writer STRING,
    INDEX owner_idx (owner),
    INDEX starts_at_idx (starts_at),
    INDEX ends_at_idx (ends_at),
    INVERTED INDEX cell_idx (cells),
    INDEX subs_by_time_with_owner (ends_at) STORING (owner),
    CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at),
    CONSTRAINT subs_cells_not_null CHECK (array_length(cells, 1) IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS identification_service_areas (
    id UUID PRIMARY KEY,
    owner STRING NOT NULL,
    url STRING NOT NULL,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL,
    cells INT64[] NOT NULL,
    writer STRING,
    INDEX owner_idx (owner),
    INDEX starts_at_idx (starts_at),
    INDEX ends_at_idx (ends_at),
    INDEX updated_at_idx (updated_at),
    INVERTED INDEX cell_idx (cells),
    CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at),
    CONSTRAINT tsa_cells_not_null CHECK (array_length(cells, 1) IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS schema_versions (
	onerow_enforcer bool PRIMARY KEY DEFAULT TRUE CHECK(onerow_enforcer),
	schema_version STRING NOT NULL
);

INSERT INTO schema_versions (schema_version) VALUES ('v1.0.0');
