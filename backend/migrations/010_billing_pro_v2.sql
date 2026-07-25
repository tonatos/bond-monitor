-- +goose Up
-- +goose StatementBegin
-- Pro tariff v2: 495 ₽/mo, 3540 ₽/yr (295 ₽/mo effective). Grandfather: renewals keep sub.amount_kopecks.
UPDATE billing_plan_versions
SET is_current = 0, code = 'pro_month_archived_v1'
WHERE id = 'pro_month_v1';

UPDATE billing_plan_versions
SET is_current = 0, code = 'pro_year_archived_v1'
WHERE id = 'pro_year_v1';

INSERT INTO billing_plan_versions (
    id, catalog_group, code, period, amount_kopecks, features_json, effective_from, is_current
) VALUES (
    'pro_month_v2',
    'pro',
    'pro_month',
    'month',
    49500,
    '["broker_credentials.write","portfolio.attach","trading_portfolio.access"]',
    '2026-07-25T00:00:00Z',
    1
) ON CONFLICT (id) DO NOTHING;

INSERT INTO billing_plan_versions (
    id, catalog_group, code, period, amount_kopecks, features_json, effective_from, is_current
) VALUES (
    'pro_year_v2',
    'pro',
    'pro_year',
    'year',
    354000,
    '["broker_credentials.write","portfolio.attach","trading_portfolio.access"]',
    '2026-07-25T00:00:00Z',
    1
) ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM billing_plan_versions WHERE id IN ('pro_month_v2', 'pro_year_v2');
UPDATE billing_plan_versions
SET is_current = 1, code = 'pro_month'
WHERE id = 'pro_month_v1';
UPDATE billing_plan_versions
SET is_current = 1, code = 'pro_year'
WHERE id = 'pro_year_v1';
-- +goose StatementEnd
