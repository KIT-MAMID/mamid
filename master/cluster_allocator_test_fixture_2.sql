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

INSERT INTO mongod_states VALUES (38, 28, false, 4);
INSERT INTO mongod_states VALUES (39, 28, false, 4);


--
-- Name: mongod_states_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('mongod_states_id_seq', 39, true);


--
-- Data for Name: msp_errors; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO msp_errors VALUES (2, 'COMM', 'Error communicating with slave.', 'Get http://test:8081/msp/status: dial tcp: lookup test: no such host');
INSERT INTO msp_errors VALUES (3, 'COMM', 'Error communicating with slave.', 'Get http://testfoo:8081/msp/status: dial tcp: lookup testfoo: no such host');
INSERT INTO msp_errors VALUES (1, 'COMM', 'Error communicating with slave.', 'Get http://test:8081/msp/status: dial tcp 172.21.228.10:8081: i/o timeout');


--
-- Data for Name: replica_sets; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO replica_sets VALUES (2, 'test', 1, 0, false);


--
-- Data for Name: risk_groups; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO risk_groups VALUES (1, 'test');
INSERT INTO risk_groups VALUES (3, 'test2');


--
-- Data for Name: slaves; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO slaves VALUES (1, 'test', 8081, 18080, 18081, false, 1, NULL, 2);
INSERT INTO slaves VALUES (2, '10.101.202.101', 8081, 18080, 18099, true, 1, NULL, NULL);
INSERT INTO slaves VALUES (3, '10.101.202.102', 8081, 18080, 18081, true, 3, NULL, NULL);
INSERT INTO slaves VALUES (4, 'testfoo', 8081, 18080, 18081, false, 3, NULL, 3);


--
-- Data for Name: mongods; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO mongods VALUES (28, 18080, 'test', NULL, NULL, 2, 2, 38, 39);


--
-- Name: mongods_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('mongods_id_seq', 28, true);


--
-- Name: msp_errors_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('msp_errors_id_seq', 3, true);


--
-- Data for Name: problems; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO problems VALUES (3, 'Slave `test` is unreachable', '', 1, '2016-08-29 00:17:45.764054', '2016-08-30 19:42:29.170176', 1, NULL, NULL);


--
-- Name: problems_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('problems_id_seq', 8, true);


--
-- Data for Name: replica_set_members; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: replica_set_members_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('replica_set_members_id_seq', 1, false);


--
-- Name: replica_sets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('replica_sets_id_seq', 2, true);


--
-- Name: risk_groups_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('risk_groups_id_seq', 6, true);


--
-- Name: slaves_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('slaves_id_seq', 4, true);


--
-- PostgreSQL database dump complete
--

