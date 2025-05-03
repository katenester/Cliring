create table if not exists clients (
                                       client_id  integer primary key,
                                       name       varchar(100) not null,
    inn        varchar(100),
    created_at timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at timestamp with time zone default CURRENT_TIMESTAMP
                             );

comment on table clients is 'Таблица для хранения информации о клиентах';
comment on column clients.client_id is 'Уникальный идентификатор клиента';
comment on column clients.name is 'Имя клиента';
comment on column clients.inn is 'ИНН';
comment on column clients.created_at is 'Дата и время создания';
comment on column clients.updated_at is 'Дата и время последнего обновления';

create table if not exists order_types (
                                           order_type_id integer primary key,
                                           name          varchar(20) not null unique
    );

comment on table order_types is 'Таблица для хранения типов заказов';
comment on column order_types.order_type_id is 'Уникальный идентификатор типа заказа';
comment on column order_types.name is 'Название типа заказа (Покупка, Кредит, Трейд-ин)';

create table if not exists deals (
                                     deal_id       integer primary key,
                                     is_completed  boolean default false,
                                     created_at    timestamp with time zone default CURRENT_TIMESTAMP,
                                     updated_at    timestamp with time zone default CURRENT_TIMESTAMP,
                                     dealership_id integer,
                                     manager_id    integer,
                                     client_id     integer references clients
);

comment on table deals is 'Таблица для хранения сделок';
comment on column deals.deal_id is 'Уникальный идентификатор сделки';
comment on column deals.is_completed is 'Флаг завершения сделки';
comment on column deals.created_at is 'Дата и время создания';
comment on column deals.updated_at is 'Дата и время последнего обновления';
comment on column deals.dealership_id is 'Идентификатор дилерского центра';
comment on column deals.manager_id is 'Идентификатор менеджера';
comment on column deals.client_id is 'Идентификатор клиента';

create index if not exists idx_deals_client_id on deals (client_id);

create table if not exists bank (
                                    bank_id   integer primary key,
                                    bank_name varchar(50) not null
    );

create table if not exists orders (
                                      order_id           integer primary key,
                                      deal_id            integer not null references deals,
                                      order_type_id      integer not null references order_types,
                                      amount             numeric(15, 2) not null check (amount > 0),
    status             varchar(20) default 'pending' check (status in ('pending', 'executed', 'cancelled')),
    created_at         timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at         timestamp with time zone default CURRENT_TIMESTAMP,
                                     need_and_orders_id integer,
                                     bank_id            integer references bank
                                     );

comment on table orders is 'Таблица для хранения заказов';
comment on column orders.order_id is 'Уникальный идентификатор заказа';
comment on column orders.deal_id is 'Идентификатор сделки';
comment on column orders.order_type_id is 'Идентификатор типа заказа (ссылка на order_types)';
comment on column orders.amount is 'Сумма заказа';
comment on column orders.status is 'Статус заказа: pending, executed, cancelled';
comment on column orders.created_at is 'Дата и время создания';
comment on column orders.updated_at is 'Дата и время последнего обновления';
comment on column orders.need_and_orders_id is 'Идентификатор потребности или заказа из сервиса Need and Orders';
comment on column orders.bank_id is 'Идентификатор банка';

create index if not exists idx_orders_deal_id on orders (deal_id);
create index if not exists idx_orders_order_type_id on orders (order_type_id);
create index if not exists idx_orders_bank_id on orders (bank_id);

create table if not exists monetary_settlements (
                                                    monetary_settlement_id integer primary key,
                                                    deal_id                integer not null references deals,
                                                    amount                 numeric(15, 2) not null check (amount > 0),
    status                 varchar(20) default 'pending' check (status in ('pending', 'executed', 'cancelled')),
    created_at             timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at             timestamp with time zone default CURRENT_TIMESTAMP,
                                         bank_id                integer references bank
                                         );

comment on table monetary_settlements is 'Таблица для хранения денежных взаиморасчетов';
comment on column monetary_settlements.monetary_settlement_id is 'Уникальный идентификатор денежного взаиморасчета';
comment on column monetary_settlements.deal_id is 'Идентификатор сделки';
comment on column monetary_settlements.amount is 'Сумма взаиморасчета';
comment on column monetary_settlements.status is 'Статус: pending, executed, cancelled';
comment on column monetary_settlements.created_at is 'Дата и время создания';
comment on column monetary_settlements.updated_at is 'Дата и время последнего обновления';
comment on column monetary_settlements.bank_id is 'Идентификатор банка';

create index if not exists idx_monetary_settlements_deal_id on monetary_settlements (deal_id);
create index if not exists idx_monetary_settlements_bank_id on monetary_settlements (bank_id);


---- create above / drop below ----

-- Откат: удаление таблиц в обратном порядке
drop table if exists monetary_settlements cascade;
drop table if exists orders cascade;
drop table if exists bank cascade;
drop table if exists deals cascade;
drop table if exists order_types cascade;
drop table if exists clients cascade;
