import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "./client";

export interface PersonFilm {
  tmdb_id: number;
  title: string;
  year: number;
  poster_path: string;
  in_library: boolean;
  movie_id?: string;
}

export interface PersonDetail {
  id: number;
  name: string;
  profile_path: string;
  known_for_department: string;
  films: PersonFilm[];
}

export function usePersonDetail(personId: number) {
  return useQuery({
    queryKey: ["people", personId],
    queryFn: () => apiFetch<PersonDetail>(`/people/${personId}`),
    enabled: personId > 0,
    staleTime: 5 * 60_000,
  });
}
