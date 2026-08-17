export declare global {
  interface Export {
    id?: string;
    type?: ExportType;
    target?: string;
    // filter?: any;
    status?: string;
    started_at?: string;
    ended_at?: string;
    file_name?: string;
    download_path?: string;
  }
}
