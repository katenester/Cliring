-- Создание таблицы deals
CREATE TABLE deals (
                       deal_id INTEGER PRIMARY KEY NOT NULL,
                       is_completed BOOLEAN,
                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                       updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE deals IS 'Таблица для хранения сделок';
COMMENT ON COLUMN deals.deal_id IS 'Уникальный идентификатор сделки';
COMMENT ON COLUMN deals.is_completed IS 'Флаг завершения сделки';
COMMENT ON COLUMN deals.created_at IS 'Дата и время создания';
COMMENT ON COLUMN deals.updated_at IS 'Дата и время последнего обновления';

-- Создание таблицы orders
CREATE TABLE orders (
                        order_id INTEGER PRIMARY KEY NOT NULL,
                        deal_id INTEGER NOT NULL,
                        amount NUMERIC(15,2) NOT NULL,
                        status CHARACTER VARYING(20) DEFAULT 'pending',
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        client_id INTEGER,
                        "counterparty-id" INTEGER,
                        need_and_orders_id INTEGER,
                        FOREIGN KEY (deal_id) REFERENCES deals (deal_id)
                            MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION
);

COMMENT ON TABLE orders IS 'Таблица для хранения взаиморасчетов типа "Заказ"';
COMMENT ON COLUMN orders.order_id IS 'Уникальный идентификатор заказа';
COMMENT ON COLUMN orders.deal_id IS 'Ссылка на сделку';
COMMENT ON COLUMN orders.amount IS 'Сумма заказа';
COMMENT ON COLUMN orders.status IS 'Текущий статус заказа (pending, executed, cancelled)';
COMMENT ON COLUMN orders.created_at IS 'Дата и время создания заказа';
COMMENT ON COLUMN orders.updated_at IS 'Дата и время последнего обновления заказа';
COMMENT ON COLUMN orders.client_id IS 'Клиент-кредитор';
COMMENT ON COLUMN orders."counterparty-id" IS 'Клиент-кредитор';
COMMENT ON COLUMN orders.need_and_orders_id IS 'ID потребности, заказа (сервис Need and Orders)';

-- Создание таблицы monetary_settlements
CREATE TABLE monetary_settlements (
                                      monetary_settlement_id INTEGER PRIMARY KEY NOT NULL,
                                      deal_id INTEGER,
                                      amount NUMERIC(15,2) NOT NULL,
                                      status CHARACTER VARYING(20) DEFAULT 'pending',
                                      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                      updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                      payment_method CHARACTER VARYING(50),
                                      client_id INTEGER,
                                      FOREIGN KEY (deal_id) REFERENCES deals (deal_id)
                                          MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION
);

COMMENT ON TABLE monetary_settlements IS 'Таблица для хранения взаиморасчетов типа "Денежный платеж"';
COMMENT ON COLUMN monetary_settlements.monetary_settlement_id IS 'Уникальный идентификатор денежного взаиморасчета';
COMMENT ON COLUMN monetary_settlements.deal_id IS 'Ссылка на сделку';
COMMENT ON COLUMN monetary_settlements.amount IS 'Сумма взаиморасчета';
COMMENT ON COLUMN monetary_settlements.status IS 'Текущий статус (pending, executed, cancelled)';
COMMENT ON COLUMN monetary_settlements.created_at IS 'Дата и время создания';
COMMENT ON COLUMN monetary_settlements.updated_at IS 'Дата и время последнего обновления';
COMMENT ON COLUMN monetary_settlements.payment_method IS 'Метод оплаты';
COMMENT ON COLUMN monetary_settlements.client_id IS 'Клиент';

-- Создание таблицы clearing_transactions
CREATE TABLE clearing_transactions (
                                       clearing_transaction_id INTEGER PRIMARY KEY NOT NULL,
                                       order_id INTEGER,
                                       monetary_settlement_id INTEGER,
                                       amount NUMERIC(15,2) NOT NULL,
                                       status CHARACTER VARYING(20) DEFAULT 'pending',
                                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                       updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                       FOREIGN KEY (monetary_settlement_id) REFERENCES monetary_settlements (monetary_settlement_id)
                                           MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION,
                                       FOREIGN KEY (order_id) REFERENCES orders (order_id)
                                           MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION
);

COMMENT ON TABLE clearing_transactions IS 'Таблица для хранения клиринговых транзакций';
COMMENT ON COLUMN clearing_transactions.clearing_transaction_id IS 'Уникальный идентификатор клиринговой транзакции';
COMMENT ON COLUMN clearing_transactions.order_id IS 'Ссылка на заказ (опционально)';
COMMENT ON COLUMN clearing_transactions.monetary_settlement_id IS 'Ссылка на денежный взаиморасчет (опционально)';
COMMENT ON COLUMN clearing_transactions.amount IS 'Сумма транзакции';
COMMENT ON COLUMN clearing_transactions.status IS 'Статус: pending, completed';
COMMENT ON COLUMN clearing_transactions.created_at IS 'Дата и время создания';
COMMENT ON COLUMN clearing_transactions.updated_at IS 'Дата и время последнего обновления';

-- Создание таблицы pay_transactions
CREATE TABLE pay_transactions (
                                  transaction_id INTEGER PRIMARY KEY NOT NULL,
                                  monetary_settlement_id INTEGER NOT NULL,
                                  amount NUMERIC(15,2) NOT NULL,
                                  bank_account CHARACTER VARYING(100) NOT NULL,
                                  payment_method CHARACTER VARYING(50) NOT NULL,
                                  status CHARACTER VARYING(20) DEFAULT 'pending',
                                  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                  FOREIGN KEY (monetary_settlement_id) REFERENCES monetary_settlements (monetary_settlement_id)
                                      MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION
);

COMMENT ON TABLE pay_transactions IS 'Таблица для хранения денежных транзакций в модуле Клиринга';
COMMENT ON COLUMN pay_transactions.transaction_id IS 'Уникальный идентификатор транзакции';
COMMENT ON COLUMN pay_transactions.monetary_settlement_id IS 'Ссылка на денежный взаиморасчет';
COMMENT ON COLUMN pay_transactions.amount IS 'Сумма транзакции';
COMMENT ON COLUMN pay_transactions.bank_account IS 'Банковский счет для транзакции';
COMMENT ON COLUMN pay_transactions.payment_method IS 'Метод оплаты';
COMMENT ON COLUMN pay_transactions.status IS 'Текущий статус (pending, completed, cancelled)';
COMMENT ON COLUMN pay_transactions.created_at IS 'Дата и время создания';
COMMENT ON COLUMN pay_transactions.updated_at IS 'Дата и время последнего обновления';

-- Создание таблицы payment_schedules
CREATE TABLE payment_schedules (
                                   schedule_id INTEGER PRIMARY KEY NOT NULL,
                                   monetary_settlement_id INTEGER NOT NULL,
                                   due_date DATE NOT NULL,
                                   amount NUMERIC(15,2) NOT NULL,
                                   status CHARACTER VARYING(20) DEFAULT 'planned',
                                   created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                   updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                   FOREIGN KEY (monetary_settlement_id) REFERENCES monetary_settlements (monetary_settlement_id)
                                       MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION
);

COMMENT ON TABLE payment_schedules IS 'Таблица для управления графиками платежей';
COMMENT ON COLUMN payment_schedules.schedule_id IS 'Уникальный идентификатор записи в графике';
COMMENT ON COLUMN payment_schedules.monetary_settlement_id IS 'Ссылка на денежный взаиморасчет';
COMMENT ON COLUMN payment_schedules.due_date IS 'Дата платежа';
COMMENT ON COLUMN payment_schedules.amount IS 'Сумма платежа';
COMMENT ON COLUMN payment_schedules.status IS 'Статус: planned, paid, overdue';
COMMENT ON COLUMN payment_schedules.created_at IS 'Дата и время создания';
COMMENT ON COLUMN payment_schedules.updated_at IS 'Дата и время последнего обновления';

-- Создание таблицы personal_wallets
CREATE TABLE personal_wallets (
                                  wallet_id INTEGER PRIMARY KEY NOT NULL,
                                  balance BYTEA NOT NULL,
                                  contract_id INTEGER NOT NULL,
                                  status CHARACTER VARYING(20) DEFAULT 'active',
                                  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                  client_id INTEGER NOT NULL
);

CREATE INDEX idx_personal_wallets_status ON personal_wallets USING BTREE (status);
CREATE UNIQUE INDEX personal_wallets_pk ON personal_wallets USING BTREE (client_id);

COMMENT ON TABLE personal_wallets IS 'Таблица для управления виртуальными кошельками клиентов';
COMMENT ON COLUMN personal_wallets.wallet_id IS 'Уникальный идентификатор кошелька';
COMMENT ON COLUMN personal_wallets.balance IS 'Текущий баланс кошелька';
COMMENT ON COLUMN personal_wallets.contract_id IS 'Ссылка на договор-основание (от сервиса Document_id)';
COMMENT ON COLUMN personal_wallets.status IS 'Текущий статус кошелька (active, frozen, closed)';
COMMENT ON COLUMN personal_wallets.created_at IS 'Дата и время создания кошелька';
COMMENT ON COLUMN personal_wallets.updated_at IS 'Дата и время последнего обновления кошелька';
COMMENT ON COLUMN personal_wallets.client_id IS 'id Клиента кошелька';

---- create above / drop below ----

-- Откат: удаление таблиц в обратном порядке
DROP TABLE personal_wallets;
DROP TABLE payment_schedules;
DROP TABLE pay_transactions;
DROP TABLE clearing_transactions;
DROP TABLE monetary_settlements;
DROP TABLE orders;
DROP TABLE deals;