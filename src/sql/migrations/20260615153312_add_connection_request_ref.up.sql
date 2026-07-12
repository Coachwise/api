-- Trace each established connection back to the request it came from.
ALTER TABLE public.connections
    ADD COLUMN connection_request_id uuid REFERENCES public.connection_requests(id) ON DELETE SET NULL;
