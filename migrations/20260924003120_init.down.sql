DROP TABLE IF EXISTS outbox_entries;
DROP TRIGGER IF EXISTS wallet_ledger_entries_no_delete ON wallet_ledger_entries;
DROP TRIGGER IF EXISTS wallet_ledger_entries_no_update ON wallet_ledger_entries;
DROP FUNCTION IF EXISTS forbid_ledger_mutation();
DROP TABLE IF EXISTS wallet_ledger_entries;
DROP TABLE IF EXISTS wager_transactions;
DROP TABLE IF EXISTS wallets;