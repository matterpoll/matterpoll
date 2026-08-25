/**
 * The actions this plugin dispatches.
 *
 * `type` is typed as `string` rather than as a literal because the runtime values are built
 * from the plugin id in plugin.json, which TypeScript only knows as `string`. The reducers
 * therefore each take the action they handle instead of narrowing a union on `action.type`.
 */

import type {PollMetadata} from '@/types/poll';

export type FetchPollMetadataAction = {
    type: string;
    data?: PollMetadata;
};

export type RegisterPostTypeComponentIdAction = {
    type: string;
    data?: {
        postTypeComponentId?: string;
    };
};
