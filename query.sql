DROP table iot_data;

CREATE table iot_data(
                         id SERIAL PRIMARY KEY,         -- 고유 ID, 자동 증가
                         humidity NUMERIC(10, 2) NOT null,
                         temperature NUMERIC(10, 2) NOT null,
                         light_quantity NUMERIC(10, 2) NOT null,
                         battery_voltage NUMERIC(10, 2) NOT null,
                         solar_voltage NUMERIC(10, 2) NOT null,
                         load_ampere NUMERIC(10, 2) NOT null,
                         timestamp TIMESTAMPTZ NOT NULL -- 타임스탬프 (타임존 포함)
);

--Index 생성
CREATE INDEX timestamp_idx ON iot_data(timestamp);
SELECT * FROM pg_indexes WHERE tablename = 'iot_data';

select * from iot_data ;
select count(*) from iot_data;
delete from iot_data;

INSERT INTO iot_data  (timestamp, humidity, temperature, light_quantity, battery_voltage, solar_voltage, load_ampere)
VALUES (NOW(), 51.9, 22.7, 762, 9.42, 0.28, 0);

commit;

SELECT timestamp, humidity, temperature, light_quantity FROM iot_data WHERE timestamp >= '2024-10-16 17:06:07.913113';


drop table docker_images ;
CREATE TABLE docker_images (
                               id SERIAL PRIMARY KEY,
                               name VARCHAR(255) NOT NULL,
                               namespace VARCHAR(255) NOT NULL,
                               description TEXT,
                               pull_count BIGINT,
                               star_count BIGINT,
                               is_private BOOLEAN,
                               last_updated TIMESTAMP,
                               media_types TEXT,
                               content_types TEXT,
                               storage_size VARCHAR(255),
                               category1 VARCHAR(255),
                               category2 VARCHAR(255),
                               category3 VARCHAR(255),
                               category4 VARCHAR(255),
                               image_type VARCHAR(50),
                               UNIQUE (name, namespace)
);

ALTER TABLE docker_images ALTER COLUMN storage_size TYPE VARCHAR(255);

delete from docker_images;

select count(*) from docker_images di ;
select * from docker_images; where image_type  = 'Verified Publisher' and name = 'ubuntu-server';