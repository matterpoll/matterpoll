import request from 'superagent';

import {id as pluginId} from '@/manifest';
import type {PollConfiguration} from '@/types/poll';

type Headers = Record<string, string | number>;
type RequestBody = string | Record<string, unknown>;

export default class Client {
    url: string;

    constructor() {
        this.url = `/plugins/${pluginId}/api/v1`;
    }

    getPluginConfiguration = async (): Promise<PollConfiguration> => {
        return this.doGet<PollConfiguration>(`${this.url}/configuration`);
    };

    doGet = async <T>(url: string, body?: RequestBody, headers: Headers = {}): Promise<T> => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const response = await request.
            get(url).
            set(headers).
            accept('application/json');

        return response.body;
    };

    doPost = async <T>(url: string, body?: RequestBody, headers: Headers = {}): Promise<T> => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const response = await request.
            post(url).
            send(body).
            set(headers).
            type('application/json').
            accept('application/json');

        return response.body;
    };

    doDelete = async <T>(url: string, body?: RequestBody, headers: Headers = {}): Promise<T> => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const response = await request.
            delete(url).
            send(body).
            set(headers).
            type('application/json').
            accept('application/json');

        return response.body;
    };

    doPut = async <T>(url: string, body?: RequestBody, headers: Headers = {}): Promise<T> => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const response = await request.
            put(url).
            send(body).
            set(headers).
            type('application/json').
            accept('application/json');

        return response.body;
    };
}
