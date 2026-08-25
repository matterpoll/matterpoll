/**
 * Shapes of the message attachment data Matterpoll's server puts on `custom_matterpoll` posts,
 * plus the poll metadata the plugin fetches for the current user.
 */

export type AttachmentField = {
    title: string;
    value: string;
    short?: boolean;
};

export type AttachmentAction = {
    id: string;
    name: string;
    type?: string;
    style?: string;
    cookie?: string;
};

export type Attachment = {
    author_name?: string;
    title?: string;
    text?: string;
    fields?: AttachmentField[];
    actions?: AttachmentAction[];
};

export type PollPost = {
    id: string;
    props: {
        attachments?: Attachment[];
        poll_id?: string;
    };
};

export type PollMetadata = {
    poll_id: string;
    voted_answers?: string[];
    can_manage_poll?: boolean;
    setting_progress?: boolean;
    setting_public_add_option?: boolean;
};

export type PollMetadataMap = Record<string, PollMetadata>;

export type DefaultSettingsValue = {
    anonymous?: boolean;
    anonymousCreator?: boolean;
    progress?: boolean;
    publicAddOption?: boolean;
};
