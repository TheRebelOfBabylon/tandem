CREATE MIGRATION m1kj427idufkn5ljuwyv4uayisknnzhkgu7t3ulnpa72gkgq6jbsuq
    ONTO m17fvi4xhkf2wpdtllg3hdmw7mtsyrnd77hqqhnu6qkutgueydn6bq
{
  ALTER TYPE events::Event {
      ALTER PROPERTY tags {
          SET TYPE array<std::json> USING (<array<std::json>>.tags);
      };
  };
};
