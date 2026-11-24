import Fastify from 'fastify';
import pg from 'pg';

pg.types.setTypeParser(20, (v) => Number(v))