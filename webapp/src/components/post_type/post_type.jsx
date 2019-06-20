import React from 'react';
import PropTypes from 'prop-types';

import LinkOnlyRenderer from 'utils/link_only_renderer';

import ActionView from './action_view';
import FieldsTable from './fields/fields_table';

const {formatText, messageHtmlToComponent} = window.PostUtils;

export default class PostType extends React.PureComponent {
    static propTypes = {
        post: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,

        options: PropTypes.object,
        postTypeComponentId: PropTypes.string,
    }

    static defaultProps = {
        options: {
            atMentions: true,
        },
    }

    render() {
        const {post} = this.props;
        const attachment = post.props.attachments[0] || {};

        const author = [];
        if (attachment.author_name) {
            author.push(
                <span
                    className='attachment__author-name'
                    key={'attachment__author-name'}
                >
                    {attachment.author_name}
                </span>
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
            <div
                className={'attachment'}
                ref='attachment'
            >
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
                                    options={this.props.options}
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
    }
}

const style = {
    footer: {clear: 'both'},
};
