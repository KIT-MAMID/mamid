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
-- Name: mongod_states_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('mongod_states_id_seq', 3, true);



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

INSERT INTO slaves VALUES (1, 'test1', 8081, 18080, 18081, true, 1, NULL, NULL);
INSERT INTO slaves VALUES (2, 'test2', 8081, 18080, 18081, false, 1, NULL, NULL);
INSERT INTO slaves VALUES (3, 'test3', 8081, 18080, 18081, false, 1, NULL, NULL);


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
