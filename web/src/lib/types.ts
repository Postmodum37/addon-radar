export interface Addon {
	id: number;
	name: string;
	slug: string;
	summary?: string;
	author_name?: string;
	logo_url?: string;
	download_count: number;
	thumbs_up_count: number;
	popularity_rank?: number;
	game_versions: string[];
	last_updated_at?: string;
}

export interface TrendingAddon extends Addon {
	score: number;
}

export interface Snapshot {
	recorded_at: string;
	download_count: number;
	thumbs_up_count?: number;
	popularity_rank?: number;
}

export interface Category {
	id: number;
	name: string;
	slug: string;
	parent_id?: number;
	icon_url?: string;
}

export interface PaginatedResponse<T> {
	data: T[];
	meta: {
		page: number;
		per_page: number;
		total: number;
		total_pages: number;
	};
}

export interface DataResponse<T> {
	data: T;
}
