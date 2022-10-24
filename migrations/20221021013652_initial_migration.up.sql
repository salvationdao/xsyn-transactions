CREATE TABLE account_codes
(
    id    INTEGER NOT NULL PRIMARY KEY,
    label TEXT    NOT NULL
);

INSERT INTO public.account_codes (id, label) VALUES (1, 'unknown');
INSERT INTO public.account_codes (id, label) VALUES (2, 'reserve');
INSERT INTO public.account_codes (id, label) VALUES (3, 'system');
INSERT INTO public.account_codes (id, label) VALUES (4, 'user');
INSERT INTO public.account_codes (id, label) VALUES (5, 'bot');


CREATE TABLE ledgers
(
    id    INTEGER NOT NULL PRIMARY KEY,
    label TEXT    NOT NULL
);

CREATE TABLE accounts
(
    id             UUID                     DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    xsyn_user_id   UUID                                               NOT NULL,
    account_code   INTEGER                  DEFAULT 1                 NOT NULL REFERENCES account_codes,
    ledger         INTEGER                  DEFAULT 0                 NOT NULL REFERENCES ledgers (id),
    debits_posted  NUMERIC(28)              DEFAULT 0                 NOT NULL,
    credits_posted NUMERIC(28)              DEFAULT 0                 NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()             NOT NULL,
    UNIQUE (xsyn_user_id, ledger)
);

INSERT INTO public.ledgers (id, label) VALUES (0, 'legacy_dont_use');
INSERT INTO public.ledgers (id, label) VALUES (1, 'sups');

CREATE TABLE transactions
(
    id                UUID                     DEFAULT gen_random_uuid() NOT NULL,
    amount            NUMERIC(28)                                        NOT NULL,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()             NOT NULL,
    debit_account_id  UUID                                               NOT NULL REFERENCES accounts (id),
    credit_account_id UUID                                               NOT NULL REFERENCES accounts (id),
    ledger            INTEGER                                            NOT NULL REFERENCES ledgers (id),
    transfer_code     INTEGER                                            NOT NULL,
    CONSTRAINT transactions_pkey2
        PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('transactions', 'created_at');

CREATE INDEX ts_transactions_created_at_descending ON transactions (created_at DESC);
CREATE INDEX ts_transactions_created_at_debit_credit_id_descending ON transactions (created_at ASC, debit_account_id ASC, credit_account_id DESC);
CREATE INDEX ts_transactions_created_at_debit_credit_id_ledger_descending ON transactions (created_at ASC, debit_account_id ASC, credit_account_id ASC, ledger DESC);
CREATE INDEX ts_transactions_credit ON transactions (credit_account_id);
CREATE INDEX ts_transactions_credit_ledger ON transactions (credit_account_id, ledger);
CREATE INDEX ts_transactions_debit ON transactions (debit_account_id);
CREATE INDEX ts_transactions_debit_ledger ON transactions (debit_account_id, ledger);
CREATE INDEX ts_transactions_debit_credit ON transactions (debit_account_id, credit_account_id);
CREATE INDEX ts_transactions_debit_credit_ledger ON transactions (debit_account_id, credit_account_id, ledger);
CREATE INDEX ts_transactions_credit_debit ON transactions (credit_account_id, debit_account_id);
CREATE INDEX ts_transactions_credit_debit_ledger ON transactions (credit_account_id, debit_account_id, ledger);

-- rewrite trigger to work with new accounts and transactions table and checks for ledger number
CREATE OR REPLACE FUNCTION check_balances() RETURNS TRIGGER AS
$check_balances$
BEGIN
    -- check its not a transaction to themselves
    IF new.debit_account_id = new.credit_account_id THEN
        RAISE EXCEPTION 'unable to transfer to self';
    END IF;

    -- check ledgers match accounts
    IF NOT (SELECT ledger = new.ledger FROM accounts WHERE id = new.debit_account_id)
    THEN
        RAISE EXCEPTION 'debit account ledger does not match transaction ledger';
    END IF;
    IF NOT (SELECT ledger = new.ledger FROM accounts WHERE id = new.credit_account_id)
    THEN
        RAISE EXCEPTION 'credit account ledger does not match transaction ledger';
    END IF;

    -- checks if the debtor is the on chain / off world account since that is the only account allow to go negative.
    IF ((SELECT xsyn_user_id != '2fa1a63e-a4fa-4618-921f-4b4d28132069' FROM accounts WHERE id = new.debit_account_id)
        AND (SELECT (accounts.credits_posted - accounts.debits_posted - new.amount) < 0
             FROM accounts
             WHERE accounts.id = new.debit_account_id
               AND accounts.ledger = new.ledger)) THEN
        RAISE EXCEPTION 'not enough funds';
    END IF;

    -- update the balances
    UPDATE accounts SET debits_posted = debits_posted + new.amount WHERE accounts.id = new.debit_account_id;
    UPDATE accounts SET credits_posted = credits_posted + new.amount WHERE accounts.id = new.credit_account_id;
    RETURN new;
END
$check_balances$
    LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_balance
    BEFORE INSERT
    ON transactions
    FOR EACH ROW
EXECUTE PROCEDURE check_balances();
