CREATE TABLE IF NOT EXISTS block_header
(
        created_at              timestamptz not null default now(),
        shard                   bigint not null,
        seqno                   bigint not null,
        gen_utime               timestamptz not null,
        root_hash               bytea not null,
        file_hash               bytea not null,
        unique (shard, seqno)
);
