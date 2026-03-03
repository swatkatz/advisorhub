-- Reseed: truncate all seeded tables so the seed loader re-runs on next startup.
-- Run manually: cat backend/migrations/014_reseed.sql | railway connect Postgres
TRUNCATE
  action_items,
  alerts,
  advisor_notes,
  goals,
  transfers,
  contributions,
  client_contribution_limits,
  accounts,
  resp_beneficiaries,
  clients,
  households,
  advisors
CASCADE;
