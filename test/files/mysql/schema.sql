-- Don't forget to maintain order here, foreign keys!
drop table if exists video_tags;
drop table if exists tags;
drop table if exists videos;
drop table if exists sponsors;
drop table if exists users;
drop table if exists type_monsters;
drop view if exists user_videos;

create table users (
	id int primary key not null auto_increment
);

create table sponsors (
	id int primary key not null auto_increment
);

create table videos (
	id int primary key not null auto_increment,

	user_id int not null COMMENT 'This is a column',
	sponsor_id int unique,

	foreign key (user_id) references users (id),
	foreign key (sponsor_id) references sponsors (id)
);

create table tags (
	id int primary key not null auto_increment
);

create table video_tags (
	video_id int not null,
	tag_id   int not null,

	primary key (video_id, tag_id),
	foreign key (video_id) references videos (id),
	foreign key (tag_id) references tags (id)
);

create table type_monsters (
	id int primary key not null auto_increment COMMENT 'This is another column',

	enum_use        enum('monday', 'tuesday', 'wednesday', 'thursday', 'friday') not null,
	enum_nullable   enum('monday', 'tuesday', 'wednesday', 'thursday', 'friday'),

	id_two     int not null,
	id_three   int,
	bool_zero  bool,
	bool_one   bool null,
	bool_two   bool not null,
	bool_three bool null default false,
	bool_four  bool null default true,
	bool_five  bool not null default false,
	bool_six   bool not null default true,

	string_zero   varchar(1),
	string_one    varchar(1) null,
	string_two    varchar(1) not null,
	string_three  varchar(1) null default 'a',
	string_four   varchar(1) not null default 'b',
	string_five   varchar(1000),
	string_six    varchar(1000) null,
	string_seven  varchar(1000) not null,
	string_eight  varchar(1000) null default 'abcdefgh',
	string_nine   varchar(1000) not null default 'abcdefgh',
	string_ten    varchar(1000) null default '',
	string_eleven varchar(1000) not null default '',

	big_int_zero  bigint,
	big_int_one   bigint NULL,
	big_int_two   bigint NOT NULL,
	big_int_three bigint NULL DEFAULT 111111,
	big_int_four  bigint NOT NULL DEFAULT 222222,
	big_int_five  bigint NULL DEFAULT 0,
	big_int_six   bigint NOT NULL DEFAULT 0,
	big_int_seven bigint UNSIGNED NOT NULL,
	big_int_eight bigint UNSIGNED NULL,

	int_zero  int,
	int_one   int NULL,
	int_two   int NOT NULL,
	int_three int NULL DEFAULT 333333,
	int_four  int NOT NULL DEFAULT 444444,
	int_five  int NULL DEFAULT 0,
	int_six   int NOT NULL DEFAULT 0,
	int_seven int UNSIGNED NOT NULL,
	int_eight int UNSIGNED NULL,

	float_zero  float,
	float_one   float,
	float_two   float(2,1),
	float_three float(2,1),
	float_four  float(2,1) null,
	float_five  float(2,1) not null,
	float_six   float(2,1) null default 1.1,
	float_seven float(2,1) not null default 1.1,
	float_eight float(2,1) null default 0.0,
	float_nine  float(2,1) null default 0.0,
	decimal_zero  decimal,
	decimal_one   decimal,
	decimal_two   decimal(2,1),
	decimal_three decimal(2,1),
	decimal_four  decimal(2,1) null,
	decimal_five  decimal(2,1) not null,
	decimal_six   decimal(2,1) null default 1.1,
	decimal_seven decimal(2,1) not null default 1.1,
	decimal_eight decimal(2,1) null default 0.0,
	decimal_nine  decimal(2,1) null default 0.0,
	bytea_zero  binary,
	bytea_one   binary null,
	bytea_two   binary not null,
	bytea_three binary not null default 'a',
	bytea_four  binary null default 'b',
	bytea_five  binary(100) not null default 'abcdefghabcdefghabcdefgh',
	bytea_six   binary(100) null default 'hgfedcbahgfedcbahgfedcba',
	bytea_seven binary not null default '',
	bytea_eight binary not null default '',
	time_zero   timestamp,
	time_one    date,
	time_two    timestamp null default null,
	time_three  timestamp null,
	time_five   timestamp null default current_timestamp,
	time_nine   timestamp not null default current_timestamp,
	time_eleven date null,
	time_twelve date not null,
	time_fifteen date null default '19990108',
	time_sixteen date not null default '1999-01-08',

	json_null  json null,
	json_nnull json not null,

	tinyint_null    tinyint null,
	tinyint_nnull   tinyint not null,
	tinyint1_null   tinyint(1) null,
	tinyint1_nnull  tinyint(1) not null,
	tinyint2_null   tinyint(2) null,
	tinyint2_nnull  tinyint(2) not null,
	smallint_null   smallint null,
	smallint_nnull  smallint not null,
	mediumint_null  mediumint null,
	mediumint_nnull mediumint not null,
	bigint_null     bigint null,
	bigint_nnull    bigint not null,

	float_null       float null,
	float_nnull      float not null,

	double_null      double null,
	double_nnull     double not null,

	decimal_null  decimal null,
	decimal_nnull decimal not null,

	real_null  real null,
	real_nnull real not null,

	boolean_null  boolean null,
	boolean_nnull boolean not null,

	date_null  date null,
	date_nnull date not null,

	datetime_null  datetime null,
	datetime_nnull datetime not null,

	timestamp_null  timestamp null,
	timestamp_nnull timestamp not null default current_timestamp,

	binary_null      binary null,
	binary_nnull     binary not null,
	varbinary_null   varbinary(100) null,
	varbinary_nnull  varbinary(100) not null,
	tinyblob_null    tinyblob null,
	tinyblob_nnull   tinyblob not null,
	blob_null        blob null,
	blob_nnull       blob not null,
	mediumblob_null  mediumblob null,
	mediumblob_nnull mediumblob not null,
	longblob_null    longblob null,
	longblob_nnull   longblob not null,

	varchar_null  varchar(100) null,
	varchar_nnull varchar(100) not null,
	char_null     char null,
	char_nnull    char not null,
	text_null     text null,
	text_nnull    text not null,


    virtual_nnull text GENERATED ALWAYS AS (UPPER(text_nnull)) VIRTUAL NOT NULL,
    virtual_null text GENERATED ALWAYS AS (UPPER(text_null)) VIRTUAL,
    generated_nnull text GENERATED ALWAYS AS (UPPER(text_nnull)) STORED NOT NULL,
    generated_null text GENERATED ALWAYS AS (UPPER(text_null)) STORED,

    UNIQUE(int_one, int_two)
) COMMENT = 'This is a table';

create view user_videos as
select u.id user_id, v.id video_id, v.sponsor_id sponsor_id
from users u
inner join videos v on v.user_id = u.id;


CREATE TABLE multi_keys (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,

	user_id INT NOT NULL,
	sponsor_id INT UNIQUE,

    something INT CHECK (something > 0 || something <= 0), -- Check constraint that is always true
    another INT,
    
  	one   int NULL,
	two   int NOT NULL,

    full_text_col LONGTEXT NOT NULL,

    UNIQUE(something, another),
    FULLTEXT INDEX idx_full_text (full_text_col),
    FOREIGN KEY (one, two) REFERENCES type_monsters(int_one, int_two)
);

CREATE TABLE test_index_expressions (
    col1 int,
    col2 int,
    col3 int
);
CREATE INDEX idx1 ON test_index_expressions ((col1 + col2)) COMMENT 'This is an index';
CREATE INDEX idx2 ON test_index_expressions ((col1 + col2), col3);
CREATE INDEX idx3 ON test_index_expressions (col1, (col2 + col3));
CREATE INDEX idx4 ON test_index_expressions (col3);
CREATE INDEX idx5 ON test_index_expressions (col1 DESC, col2 DESC);
CREATE INDEX idx6 ON test_index_expressions ((POW(col3, 2)));

CREATE TABLE foo_bar (
    id INT AUTO_INCREMENT PRIMARY KEY,
    secret_col VARCHAR(255) NOT NULL
);
CREATE TABLE foo_baz (
    id INT AUTO_INCREMENT PRIMARY KEY,
    secret_col VARCHAR(255) NOT NULL
);
CREATE TABLE foo_qux (
    id INT AUTO_INCREMENT PRIMARY KEY,
    secret_col VARCHAR(255) NOT NULL
);
CREATE TABLE bar_baz (
    id INT AUTO_INCREMENT PRIMARY KEY,
    secret_col VARCHAR(255) NOT NULL
);
CREATE TABLE bar_qux (
    id INT AUTO_INCREMENT PRIMARY KEY,
    secret_col VARCHAR(255) NOT NULL
);

CREATE TABLE query (
    id INT AUTO_INCREMENT PRIMARY KEY,
    query_text TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

