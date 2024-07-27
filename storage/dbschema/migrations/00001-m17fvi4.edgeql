CREATE MIGRATION m17fvi4xhkf2wpdtllg3hdmw7mtsyrnd77hqqhnu6qkutgueydn6bq
    ONTO initial
{
  CREATE MODULE events IF NOT EXISTS;
  CREATE TYPE events::Event {
      CREATE PROPERTY content: std::str;
      CREATE REQUIRED PROPERTY createdAt: std::datetime;
      CREATE REQUIRED PROPERTY eventId: std::str {
          CREATE CONSTRAINT std::exclusive;
      };
      CREATE REQUIRED PROPERTY kind: std::int64;
      CREATE REQUIRED PROPERTY pubkey: std::str;
      CREATE REQUIRED PROPERTY sig: std::str {
          CREATE CONSTRAINT std::exclusive;
      };
      CREATE PROPERTY tags: std::json;
  };
};
