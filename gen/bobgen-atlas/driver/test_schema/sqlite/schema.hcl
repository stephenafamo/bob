table "users" {
  schema = schema.main
  column "id" {
    null = false
    type = int
  }
  primary_key {
    columns = [column.id]
  }
}
table "sponsors" {
  schema = schema.main
  column "id" {
    null = false
    type = int
  }
  primary_key {
    columns = [column.id]
  }
}
table "videos" {
  schema = schema.main
  column "id" {
    null = false
    type = int
  }
  column "user_id" {
    null = false
    type = int
  }
  column "sponsor_id" {
    null = true
    type = int
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "0" {
    columns     = [column.sponsor_id]
    ref_columns = [table.sponsors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "1" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "sqlite_autoindex_videos_2" {
    unique  = true
    columns = [column.sponsor_id]
  }
}
table "tags" {
  schema = schema.main
  column "id" {
    null = false
    type = int
  }
  primary_key {
    columns = [column.id]
  }
}
table "video_tags" {
  schema = schema.main
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
  foreign_key "0" {
    columns     = [column.tag_id]
    ref_columns = [table.tags.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "1" {
    columns     = [column.video_id]
    ref_columns = [table.videos.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
table "type_monsters" {
  schema = schema.main
  column "id" {
    null = false
    type = int
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
    default = false
  }
  column "bool_four" {
    null    = true
    type    = bool
    default = true
  }
  column "bool_five" {
    null    = false
    type    = bool
    default = false
  }
  column "bool_six" {
    null    = false
    type    = bool
    default = true
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
  column "float_zero" {
    null = true
    type = float
  }
  column "float_one" {
    null = true
    type = float
  }
  column "float_two" {
    null = true
    type = float
  }
  column "float_three" {
    null = true
    type = float
  }
  column "float_four" {
    null = true
    type = float
  }
  column "float_five" {
    null = false
    type = float
  }
  column "float_six" {
    null    = true
    type    = float
    default = 1.1
  }
  column "float_seven" {
    null    = false
    type    = float
    default = 1.1
  }
  column "float_eight" {
    null    = true
    type    = float
    default = 0
  }
  column "float_nine" {
    null    = true
    type    = float
    default = 0
  }
  column "bytea_zero" {
    null = true
    type = sql("binary")
  }
  column "bytea_one" {
    null = true
    type = sql("binary")
  }
  column "bytea_two" {
    null = false
    type = sql("binary")
  }
  column "bytea_three" {
    null    = false
    type    = sql("binary")
    default = "a"
  }
  column "bytea_four" {
    null    = true
    type    = sql("binary")
    default = "b"
  }
  column "bytea_five" {
    null    = false
    type    = sql("binary")
    default = "abcdefghabcdefghabcdefgh"
  }
  column "bytea_six" {
    null    = true
    type    = sql("binary")
    default = "hgfedcbahgfedcbahgfedcba"
  }
  column "bytea_seven" {
    null    = false
    type    = sql("binary")
    default = ""
  }
  column "bytea_eight" {
    null    = false
    type    = sql("binary")
    default = ""
  }
  column "time_zero" {
    null = true
    type = sql("timestamp")
  }
  column "time_one" {
    null = true
    type = date
  }
  column "time_two" {
    null    = true
    type    = sql("timestamp")
    default = sql("null")
  }
  column "time_three" {
    null = true
    type = sql("timestamp")
  }
  column "time_five" {
    null    = true
    type    = sql("timestamp")
    default = sql("current_timestamp")
  }
  column "time_nine" {
    null    = false
    type    = sql("timestamp")
    default = sql("current_timestamp")
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
    default = "19990108"
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
    type = tinyint
  }
  column "tinyint1_nnull" {
    null = false
    type = tinyint
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
    type = double_precision
  }
  column "doubleprec_nnull" {
    null = false
    type = double_precision
  }
  column "real_null" {
    null = true
    type = real
  }
  column "real_nnull" {
    null = false
    type = real
  }
  column "boolean_null" {
    null = true
    type = boolean
  }
  column "boolean_nnull" {
    null = false
    type = boolean
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
    type = sql("timestamp")
  }
  column "timestamp_nnull" {
    null    = false
    type    = sql("timestamp")
    default = sql("current_timestamp")
  }
  column "binary_null" {
    null = true
    type = sql("binary")
  }
  column "binary_nnull" {
    null = false
    type = sql("binary")
  }
  column "varbinary_null" {
    null = true
    type = sql("varbinary")
  }
  column "varbinary_nnull" {
    null = false
    type = sql("varbinary")
  }
  column "tinyblob_null" {
    null = true
    type = sql("tinyblob")
  }
  column "tinyblob_nnull" {
    null = false
    type = sql("tinyblob")
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
    type = sql("mediumblob")
  }
  column "mediumblob_nnull" {
    null = false
    type = sql("mediumblob")
  }
  column "longblob_null" {
    null = true
    type = sql("longblob")
  }
  column "longblob_nnull" {
    null = false
    type = sql("longblob")
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
    type = sql("char")
  }
  column "char_nnull" {
    null = false
    type = sql("char")
  }
  column "text_null" {
    null = true
    type = text
  }
  column "text_nnull" {
    null = false
    type = text
  }
  primary_key {
    columns = [column.id]
  }
}
table "autoinctest" {
  schema = schema.main
  column "id" {
    type = integer
  }
  primary_key {
    columns = [column.id]
  }
}
table "autoinckeywordtest" {
  schema = schema.main
  column "id" {
    type           = integer
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
    type = text
  }
  column "another" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "0" {
    columns     = [column.user_id, column.sponsor_id]
    ref_columns = [table.videos.column.user_id, table.videos.column.sponsor_id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "sqlite_autoindex_autoinckeywordtest_1" {
    unique  = true
    columns = [column.sponsor_id]
  }
  index "sqlite_autoindex_autoinckeywordtest_2" {
    unique  = true
    columns = [column.something, column.another]
  }
}
table "has_generated_columns" {
  schema = schema.main
  column "a" {
    type = integer
  }
  column "b" {
    null = true
    type = int
  }
  column "c" {
    null = true
    type = text
  }
  column "d" {
    null = true
    type = int
    as {
      expr = "(a*abs(b))"
      type = VIRTUAL
    }
  }
  column "e" {
    null = true
    type = text
    as {
      expr = "(substr(c,b,b+1))"
      type = STORED
    }
  }
  primary_key {
    columns = [column.a]
  }
}
schema "main" {
}
