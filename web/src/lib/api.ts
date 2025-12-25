import { env } from '$env/dynamic/private';
import type {
	Addon,
	TrendingAddon,
	Snapshot,
	PaginatedResponse,
	DataResponse
} from './types';

// Server-side: use API_URL from Railway internal network
// Client-side fallback: use VITE_API_URL (only for dev)
const API_URL = env.API_URL || import.meta.env.VITE_API_URL || 'http://localhost:8080';

async function fetchApi<T>(path: string): Promise<T | null> {
	try {
		const res = await fetch(`${API_URL}${path}`);
		if (!res.ok) {
			const body = await res.text().catch(() => '');
			console.error(`API error: ${path}`, {
				status: res.status,
				statusText: res.statusText,
				body: body.slice(0, 200)
			});
			return null;
		}
		return res.json();
	} catch (error) {
		console.error(`API fetch failed: ${path}`, error);
		return null;
	}
}

export async function getTrendingHot(
	page = 1,
	perPage = 20
): Promise<PaginatedResponse<TrendingAddon> | null> {
	const params = new URLSearchParams({
		page: String(page),
		per_page: String(perPage)
	});
	return fetchApi<PaginatedResponse<TrendingAddon>>(`/api/v1/trending/hot?${params}`);
}

export async function getTrendingRising(
	page = 1,
	perPage = 20
): Promise<PaginatedResponse<TrendingAddon> | null> {
	const params = new URLSearchParams({
		page: String(page),
		per_page: String(perPage)
	});
	return fetchApi<PaginatedResponse<TrendingAddon>>(`/api/v1/trending/rising?${params}`);
}

export async function getAddon(slug: string): Promise<Addon | null> {
	const res = await fetchApi<DataResponse<Addon>>(`/api/v1/addons/${slug}`);
	return res?.data ?? null;
}

export async function getAddonHistory(slug: string, limit = 168): Promise<Snapshot[]> {
	const res = await fetchApi<DataResponse<Snapshot[]>>(
		`/api/v1/addons/${slug}/history?limit=${limit}`
	);
	return res?.data ?? [];
}

export async function searchAddons(
	query: string,
	page = 1,
	perPage = 20
): Promise<PaginatedResponse<Addon> | null> {
	const params = new URLSearchParams({
		search: query,
		page: String(page),
		per_page: String(perPage)
	});
	return fetchApi<PaginatedResponse<Addon>>(`/api/v1/addons?${params}`);
}
