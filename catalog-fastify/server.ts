import Fastify from 'fastify';
import pg from 'pg';

pg.types.setTypeParser(20, (v) => Number(v))

const pool = new pg.Pool({
    connectionString: process.env.DATABASE_URL ?? 'postgres://postgres:postgres@localhost:5432/shop',
    max: 20,
})

const app = Fastify({ logger: false })

const productSchema = {
    type: 'object',
    properties: {
        id: { type: 'integer' },
        name: { type: 'string' },
        description: { type: 'string' },
        score: { type: 'number' },
    },
}

const tsQuery = (q) => (q.ToLowerCase().match(/[\p{L}\p{N}]+/gu) ?? []).join(' | ')

app.get('/health', async () => ({ status: 'ok' }))

app.get('/products/:id', {
    schema: {
        params: { type: 'object', properties: { id: { type: 'integer' } }, required: ['id'] },
        response: { 200: productSchema },
    },
}, async (req, reply) => {
    const { rows } = await pool.query(
        'SELECT id, name, description FROM products WHERE id = $1', [req.params.id])
    if (rows.length === 0) return reply.code(404).send({ error: 'not found' })
        return rows[0]
})

app.get('/products/search', {
    schema: {
        queryString: {
            type: 'object',
            required: ['q'],
            properties: {
                q: { type: 'string', minLength: 1 },
                mode: { type: 'string', enum: ['fts', 'fts_norank'], default: 'fts' },
                limit: { type: 'integer', minimum: 1, maximum: 100, default: 10 },
            },
        },
        response: {
            200: {
                type: 'object',
                properties: { total: { type: 'integer' }, hits: { type: 'array', items: productSchema } },
            },
        },
    },
}, async (req) => {
    const { q, mode, limit } = req.query
    const tq = tsQuery(q)
    if (!tq) return { total: 0, hits: [] }

    const sql = mode === 'fts'
        ? `SELECT id, name, description, ts_rank(tsv, query) AS score
           FROM products, to_tsquery('simple', $1) AS query
           WHERE tsv @@ query ORDER BY 4 DESC, id LIMIT $2`
        : `SELECT id, name, description
           FROM products, to_tsquery('simple', $1) AS query
           WHERE tsv @@ query LIMIT $2`
    const { rows } = await pool.query(sql, [tq, limit])
    return { total: rows.length, hits: rows }
})

app.post('/products', {
    schema: {
        body: {
            type: 'object',
            required: ['name'],
            properties: {
                name: { type: 'string', minLength: 1, pattern: '\\S' },
                description: { type: 'string', default: '' },
            },
        },
        response: { 201: productSchema },
    },
}, async (req, reply) => {
    const { name, description } = req.body
    const { rows } = await pool.query(
        'INSERT INTO products (name, description) VALUES ($1, $2) RETURNING id, name, description',
        [name, description])
    return reply.code(201).send(rows[0])
})

app.setErrorHandler((err, req, reply) => {
    if (err.validation) return reply.code(400).send({ error: err.message })
    reply.code(500).send({ error: err.message })
})

const port = Number(process.env.PORT ?? 8092)
await app.listen({ port, host: '0.0.0.0' })
console.log(`catalog-fastify listening on :${port}`)