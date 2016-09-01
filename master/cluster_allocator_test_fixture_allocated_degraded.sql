--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.4
-- Dumped by pg_dump version 9.5.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

SET search_path = public, pg_catalog;

--
-- Data for Name: mamid_metadata; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO mamid_metadata VALUES ('schema_version', '0.0.1');


--
-- Data for Name: mongod_states; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO mongod_states VALUES (1, 1, 'configsvr', 5);
INSERT INTO mongod_states VALUES (2, 2, 'configsvr', 5);
INSERT INTO mongod_states VALUES (3, 3, 'configsvr', 5);


--
-- Name: mongod_states_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('mongod_states_id_seq', 3, true);


--
-- Data for Name: msp_errors; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO msp_errors VALUES (1, 'COMM', 'Error communicating with slave.', 'Get https://test1:8081/msp/status: dial tcp: lookup test1: no such host');
INSERT INTO msp_errors VALUES (3, 'COMM', 'Error communicating with slave.', 'Get https://test2:8081/msp/status: dial tcp: lookup test2: no such host');
INSERT INTO msp_errors VALUES (2, 'COMM', 'Error communicating with slave.', 'Get https://test3:8081/msp/status: dial tcp: lookup test3: no such host');
INSERT INTO msp_errors VALUES (4, 'COMM', 'Error communicating with slave.', 'Get https://test1:8081/msp/status: dial tcp: lookup test1: no such host');
INSERT INTO msp_errors VALUES (5, 'COMM', 'Error communicating with slave.', 'Get https://test2:8081/msp/status: dial tcp: lookup test2: no such host');


--
-- Data for Name: replica_sets; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO replica_sets VALUES (1, 'test', 1, 2, 'configsvr', false);


--
-- Data for Name: risk_groups; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: slaves; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO slaves VALUES (1, 'test1', 8081, 18080, 18081, true, 1, NULL, 4);
INSERT INTO slaves VALUES (2, 'test2', 8081, 18080, 18081, false, 1, NULL, 5);
INSERT INTO slaves VALUES (3, 'test3', 8081, 18080, 18081, false, 1, NULL, NULL);


--
-- Data for Name: mongods; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO mongods VALUES (1, 18080, 'test', NULL, NULL, 1, 1, 1, NULL);
INSERT INTO mongods VALUES (2, 18080, 'test', NULL, NULL, 2, 1, 2, NULL);
INSERT INTO mongods VALUES (3, 18080, 'test', NULL, NULL, 3, 1, 3, NULL);


--
-- Name: mongods_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('mongods_id_seq', 3, true);


--
-- Name: msp_errors_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('msp_errors_id_seq', 5, true);


--
-- Data for Name: problems; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO problems VALUES (3, 'Slave `test1` is unreachable - Error communicating with slave.', 'Get https://test1:8081/msp/status: dial tcp: lookup test1: no such host', 1, '2016-09-01 22:53:27.003754', '2016-09-01 22:53:27.003751', 1, NULL, NULL);
INSERT INTO problems VALUES (4, 'Slave `test2` is unreachable - Error communicating with slave.', 'Get https://test2:8081/msp/status: dial tcp: lookup test2: no such host', 1, '2016-09-01 22:53:27.009727', '2016-09-01 22:53:27.009721', 2, NULL, NULL);
INSERT INTO problems VALUES (5, 'Replica Set `test` is degraded', 'One or more Mongods in this Replica Set are not running (0/1 persistent, 0/1 volatile).', 4, '2016-09-01 22:53:27.014647', '2016-09-01 22:53:27.014644', NULL, 1, NULL);


--
-- Name: problems_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('problems_id_seq', 5, true);


--
-- Name: replica_sets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('replica_sets_id_seq', 1, true);


--
-- Name: risk_groups_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('risk_groups_id_seq', 1, false);


--
-- Name: slaves_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('slaves_id_seq', 3, true);


--
-- PostgreSQL database dump complete
--

