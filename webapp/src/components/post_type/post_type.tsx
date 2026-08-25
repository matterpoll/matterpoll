import React from 'react';

import ActionView from '@/components/post_type/action_view';
import FieldsTable from '@/components/post_type/fields/fields_table';
import type {FormatTextOptions} from '@/types/mattermost-webapp';
import type {Attachment, PollPost} from '@/types/poll';
import LinkOnlyRenderer from '@/utils/link_only_renderer';

const {formatText, messageHtmlToComponent} = window.PostUtils;

type Props = {
    post: PollPost;
    theme: Record<string, string>;
    options?: FormatTextOptions;
    postTypeComponentId?: string;
};

// React 19 drops defaultProps for function components, so this is a default parameter instead.
const DEFAULT_OPTIONS: FormatTextOptions = {
    atMentions: true,
};

const PostType = ({post, options = DEFAULT_OPTIONS}: Props) => {
    const attachment: Attachment = post.props.attachments?.[0] || {};

    const author = [];
    if (attachment.author_name) {
        author.push(
            <span
                className='attachment__author-name'
                key={'attachment__author-name'}
            >
                {attachment.author_name}
            </span>,
        );
    }

    let title;
    if (attachment.title) {
        const htmlFormattedText = formatText(attachment.title, {
            mentionHighlight: false,
            renderer: new LinkOnlyRenderer(),
            autoLinkedUrlSchemes: [],
        });
        const attachmentTitle = messageHtmlToComponent(htmlFormattedText, false, {emoji: true});
        title = (
            <h1 className='attachment__title'>
                {attachmentTitle}
            </h1>
        );
    }

    let attachmentText;
    if (attachment.text) {
        attachmentText = messageHtmlToComponent(formatText(attachment.text));
    }

    return (
        <div className={'attachment'}>
            <div className='attachment__content'>
                <div
                    className='clearfix attachment__container'
                >
                    {author}
                    {title}
                    <div>
                        <div
                            className={'attachment__body attachment__body--no_thumb'}
                        >
                            {attachmentText}
                            <FieldsTable
                                attachment={attachment}
                                options={options}
                            />
                            <ActionView
                                post={post}
                                attachment={attachment}
                            />
                        </div>
                        <div style={style.footer}/>
                    </div>
                </div>
            </div>
        </div>
    );
};

const style = {
    footer: {clear: 'both'} as React.CSSProperties,
};

// Preserves the shallow prop comparison the previous PureComponent gave.
export default React.memo(PostType);
