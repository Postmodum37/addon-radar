import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { searchAddons } from '$lib/api';

export const GET: RequestHandler = async ({ url }) => {
	const query = url.searchParams.get('q') || '';

	if (query.length < 2) {
		return json([]);
	}

	const result = await searchAddons(query, 1, 5);
	return json(result?.data ?? []);
};
