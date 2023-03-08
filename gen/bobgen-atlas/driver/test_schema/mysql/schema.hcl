table "multi_keys" {
  schema = schema.bob_droppable
  column "id" {
    null           = false
    type           = int
    auto_increment = true
  }
  column "user_id" {
    null = false
    type = int
  }
  column "sponsor_id" {
    null = true
    type = int
  }
  column "something" {
    null = true
    type = int
  }
  column "another" {
    null = true
    type = int
  }
  column "one" {
    null = true
    type = int
  }
  column "two" {
    null = false
    type = int
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "multi_keys_ibfk_1" {
    columns     = [column.one, column.two]
    ref_columns = [table.type_monsters.column.int_one, table.type_monsters.column.int_two]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "one" {
    columns = [column.one, column.two]
  }
  index "something" {
    unique  = true
    columns = [column.something, column.another]
  }
  index "sponsor_id" {
    unique  = true
    columns = [column.sponsor_id]
  }
}
table "sponsors" {
  schema = schema.bob_droppable
  column "id" {
    null           = false
    type           = int
    auto_increment = true
  }
  primary_key {
    columns = [column.id]
  }
}
table "tags" {
  schema = schema.bob_droppable
  column "id" {
    null           = false
    type           = int
    auto_increment = true
  }
  primary_key {
    columns = [column.id]
  }
}
table "type_monsters" {
  schema = schema.bob_droppable
  column "id" {
    null           = false
    type           = int
    comment        = "comment on ID"
    auto_increment = true
  }
  column "enum_use" {
    null = false
    type = enum("monday","tuesday","wednesday","thursday","friday")
  }
  column "enum_nullable" {
    null = true
    type = enum("monday","tuesday","wednesday","thursday","friday")
  }
  column "id_two" {
    null = false
    type = int
  }
  column "id_three" {
    null = true
    type = int
  }
  column "bool_zero" {
    null = true
    type = bool
  }
  column "bool_one" {
    null = true
    type = bool
  }
  column "bool_two" {
    null = false
    type = bool
  }
  column "bool_three" {
    null    = true
    type    = bool
    default = 0
  }
  column "bool_four" {
    null    = true
    type    = bool
    default = 1
  }
  column "bool_five" {
    null    = false
    type    = bool
    default = 0
  }
  column "bool_six" {
    null    = false
    type    = bool
    default = 1
  }
  column "string_zero" {
    null = true
    type = varchar(1)
  }
  column "string_one" {
    null = true
    type = varchar(1)
  }
  column "string_two" {
    null = false
    type = varchar(1)
  }
  column "string_three" {
    null    = true
    type    = varchar(1)
    default = "a"
  }
  column "string_four" {
    null    = false
    type    = varchar(1)
    default = "b"
  }
  column "string_five" {
    null = true
    type = varchar(1000)
  }
  column "string_six" {
    null = true
    type = varchar(1000)
  }
  column "string_seven" {
    null = false
    type = varchar(1000)
  }
  column "string_eight" {
    null    = true
    type    = varchar(1000)
    default = "abcdefgh"
  }
  column "string_nine" {
    null    = false
    type    = varchar(1000)
    default = "abcdefgh"
  }
  column "string_ten" {
    null    = true
    type    = varchar(1000)
    default = ""
  }
  column "string_eleven" {
    null    = false
    type    = varchar(1000)
    default = ""
  }
  column "big_int_zero" {
    null = true
    type = bigint
  }
  column "big_int_one" {
    null = true
    type = bigint
  }
  column "big_int_two" {
    null = false
    type = bigint
  }
  column "big_int_three" {
    null    = true
    type    = bigint
    default = 111111
  }
  column "big_int_four" {
    null    = false
    type    = bigint
    default = 222222
  }
  column "big_int_five" {
    null    = true
    type    = bigint
    default = 0
  }
  column "big_int_six" {
    null    = false
    type    = bigint
    default = 0
  }
  column "big_int_seven" {
    null     = false
    type     = bigint
    unsigned = true
  }
  column "big_int_eight" {
    null     = true
    type     = bigint
    unsigned = true
  }
  column "int_zero" {
    null = true
    type = int
  }
  column "int_one" {
    null = true
    type = int
  }
  column "int_two" {
    null = false
    type = int
  }
  column "int_three" {
    null    = true
    type    = int
    default = 333333
  }
  column "int_four" {
    null    = false
    type    = int
    default = 444444
  }
  column "int_five" {
    null    = true
    type    = int
    default = 0
  }
  column "int_six" {
    null    = false
    type    = int
    default = 0
  }
  column "int_seven" {
    null     = false
    type     = int
    unsigned = true
  }
  column "int_eight" {
    null     = true
    type     = int
    unsigned = true
  }
  column "float_zero" {
    null = true
    type = float
  }
  column "float_one" {
    null = true
    type = float
  }
  column "float_two" {
    null     = true
    type     = float(2)
    unsigned = false
  }
  column "float_three" {
    null     = true
    type     = float(2)
    unsigned = false
  }
  column "float_four" {
    null     = true
    type     = float(2)
    unsigned = false
  }
  column "float_five" {
    null     = false
    type     = float(2)
    unsigned = false
  }
  column "float_six" {
    null     = true
    type     = float(2)
    default  = 1.1
    unsigned = false
  }
  column "float_seven" {
    null     = false
    type     = float(2)
    default  = 1.1
    unsigned = false
  }
  column "float_eight" {
    null     = true
    type     = float(2)
    default  = 0
    unsigned = false
  }
  column "float_nine" {
    null     = true
    type     = float(2)
    default  = 0
    unsigned = false
  }
  column "bytea_zero" {
    null = true
    type = binary(1)
  }
  column "bytea_one" {
    null = true
    type = binary(1)
  }
  column "bytea_two" {
    null = false
    type = binary(1)
  }
  column "bytea_three" {
    null    = false
    type    = binary(1)
    default = sql("0x61")
  }
  column "bytea_four" {
    null    = true
    type    = binary(1)
    default = sql("0x62")
  }
  column "bytea_five" {
    null    = false
    type    = binary(100)
    default = sql("0x616263646566676861626364656667686162636465666768")
  }
  column "bytea_six" {
    null    = true
    type    = binary(100)
    default = sql("0x686766656463626168676665646362616867666564636261")
  }
  column "bytea_seven" {
    null    = false
    type    = binary(1)
    default = "0x"
  }
  column "bytea_eight" {
    null    = false
    type    = binary(1)
    default = "0x"
  }
  column "time_zero" {
    null = true
    type = timestamp
  }
  column "time_one" {
    null = true
    type = date
  }
  column "time_two" {
    null = true
    type = timestamp
  }
  column "time_three" {
    null = true
    type = timestamp
  }
  column "time_five" {
    null    = true
    type    = timestamp
    default = sql("CURRENT_TIMESTAMP")
  }
  column "time_nine" {
    null    = false
    type    = timestamp
    default = sql("CURRENT_TIMESTAMP")
  }
  column "time_eleven" {
    null = true
    type = date
  }
  column "time_twelve" {
    null = false
    type = date
  }
  column "time_fifteen" {
    null    = true
    type    = date
    default = "1999-01-08"
  }
  column "time_sixteen" {
    null    = false
    type    = date
    default = "1999-01-08"
  }
  column "json_null" {
    null = true
    type = json
  }
  column "json_nnull" {
    null = false
    type = json
  }
  column "tinyint_null" {
    null = true
    type = tinyint
  }
  column "tinyint_nnull" {
    null = false
    type = tinyint
  }
  column "tinyint1_null" {
    null = true
    type = bool
  }
  column "tinyint1_nnull" {
    null = false
    type = bool
  }
  column "tinyint2_null" {
    null = true
    type = tinyint
  }
  column "tinyint2_nnull" {
    null = false
    type = tinyint
  }
  column "smallint_null" {
    null = true
    type = smallint
  }
  column "smallint_nnull" {
    null = false
    type = smallint
  }
  column "mediumint_null" {
    null = true
    type = mediumint
  }
  column "mediumint_nnull" {
    null = false
    type = mediumint
  }
  column "bigint_null" {
    null = true
    type = bigint
  }
  column "bigint_nnull" {
    null = false
    type = bigint
  }
  column "float_null" {
    null = true
    type = float
  }
  column "float_nnull" {
    null = false
    type = float
  }
  column "double_null" {
    null = true
    type = double
  }
  column "double_nnull" {
    null = false
    type = double
  }
  column "doubleprec_null" {
    null = true
    type = double
  }
  column "doubleprec_nnull" {
    null = false
    type = double
  }
  column "real_null" {
    null = true
    type = double
  }
  column "real_nnull" {
    null = false
    type = double
  }
  column "boolean_null" {
    null = true
    type = bool
  }
  column "boolean_nnull" {
    null = false
    type = bool
  }
  column "date_null" {
    null = true
    type = date
  }
  column "date_nnull" {
    null = false
    type = date
  }
  column "datetime_null" {
    null = true
    type = datetime
  }
  column "datetime_nnull" {
    null = false
    type = datetime
  }
  column "timestamp_null" {
    null = true
    type = timestamp
  }
  column "timestamp_nnull" {
    null    = false
    type    = timestamp
    default = sql("CURRENT_TIMESTAMP")
  }
  column "binary_null" {
    null = true
    type = binary(1)
  }
  column "binary_nnull" {
    null = false
    type = binary(1)
  }
  column "varbinary_null" {
    null = true
    type = varbinary(100)
  }
  column "varbinary_nnull" {
    null = false
    type = varbinary(100)
  }
  column "tinyblob_null" {
    null = true
    type = tinyblob
  }
  column "tinyblob_nnull" {
    null = false
    type = tinyblob
  }
  column "blob_null" {
    null = true
    type = blob
  }
  column "blob_nnull" {
    null = false
    type = blob
  }
  column "mediumblob_null" {
    null = true
    type = mediumblob
  }
  column "mediumblob_nnull" {
    null = false
    type = mediumblob
  }
  column "longblob_null" {
    null = true
    type = longblob
  }
  column "longblob_nnull" {
    null = false
    type = longblob
  }
  column "varchar_null" {
    null = true
    type = varchar(100)
  }
  column "varchar_nnull" {
    null = false
    type = varchar(100)
  }
  column "char_null" {
    null = true
    type = char(1)
  }
  column "char_nnull" {
    null = false
    type = char(1)
  }
  column "text_null" {
    null = true
    type = text
  }
  column "text_nnull" {
    null = false
    type = text
  }
  column "virtual_nnull" {
    null = false
    type = text
    as {
      expr = "upper(`text_nnull`)"
      type = VIRTUAL
    }
  }
  column "virtual_null" {
    null = true
    type = text
    as {
      expr = "upper(`text_null`)"
      type = VIRTUAL
    }
  }
  column "generated_nnull" {
    null = false
    type = text
    as {
      expr = "upper(`text_nnull`)"
      type = STORED
    }
  }
  column "generated_null" {
    null = true
    type = text
    as {
      expr = "upper(`text_null`)"
      type = STORED
    }
  }
  primary_key {
    columns = [column.id]
  }
  index "int_one" {
    columns = [column.int_one, column.int_two]
  }
}
table "user_videos" {
  schema  = schema.bob_droppable
  comment = "VIEW"
  column "user_id" {
    null    = false
    type    = int
    default = 0
  }
  column "video_id" {
    null    = false
    type    = int
    default = 0
  }
  column "sponsor_id" {
    null = true
    type = int
  }
}
table "users" {
  schema = schema.bob_droppable
  column "id" {
    null           = false
    type           = int
    auto_increment = true
  }
  primary_key {
    columns = [column.id]
  }
}
table "video_tags" {
  schema = schema.bob_droppable
  column "video_id" {
    null = false
    type = int
  }
  column "tag_id" {
    null = false
    type = int
  }
  primary_key {
    columns = [column.video_id, column.tag_id]
  }
  foreign_key "video_tags_ibfk_1" {
    columns     = [column.video_id]
    ref_columns = [table.videos.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "video_tags_ibfk_2" {
    columns     = [column.tag_id]
    ref_columns = [table.tags.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "tag_id" {
    columns = [column.tag_id]
  }
}
table "videos" {
  schema = schema.bob_droppable
  column "id" {
    null           = false
    type           = int
    auto_increment = true
  }
  column "user_id" {
    null    = false
    type    = int
    comment = "this is a comment"
  }
  column "sponsor_id" {
    null = true
    type = int
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "videos_ibfk_1" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "videos_ibfk_2" {
    columns     = [column.sponsor_id]
    ref_columns = [table.sponsors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "sponsor_id" {
    unique  = true
    columns = [column.sponsor_id]
  }
  index "user_id" {
    columns = [column.user_id]
  }
}
schema "bob_droppable" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
