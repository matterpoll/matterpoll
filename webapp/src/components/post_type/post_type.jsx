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

        actions: PropTypes.shape({
            doPostAction: PropTypes.func.isRequired,
        }).isRequired,
    }

    static defaultProps = {
        options: {
            atMentions: true,
        },
    }

    isUrlSafe = (url) => {
        let unescaped;

        try {
            unescaped = decodeURIComponent(url);
        } catch (e) {
            unescaped = unescape(url);
        }

        unescaped = unescaped.replace(/[^\w:]/g, '').toLowerCase();

        return !unescaped.startsWith('javascript:') && // eslint-disable-line no-script-url
            !unescaped.startsWith('vbscript:') &&
            !unescaped.startsWith('data:');
    }

    render() {
        const {post} = this.props;
        const attachment = post.props.attachments[0] || {};

        let author = [];
        if (attachment.author_name || attachment.author_icon) {
            if (attachment.author_icon) {
                author.push(
                    <img
                        className='attachment__author-icon'
                        src={attachment.author_icon}
                        key={'attachment__author-icon'}
                        height='14'
                        width='14'
                    />
                );
            }
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
        }
        if (attachment.author_link && this.isUrlSafe(attachment.author_link)) {
            author = (
                <a
                    href={attachment.author_link}
                    target='_blank'
                    rel='noopener noreferrer'
                >
                    {author}
                </a>
            );
        }

        let title;
        if (attachment.title) {
            if (attachment.title_link && this.isUrlSafe(attachment.title_link)) {
                title = (
                    <h1 className='attachment__title'>
                        <a
                            className='attachment__title-link'
                            href={attachment.title_link}
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            {attachment.title}
                        </a>
                    </h1>
                );
            } else {
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
        }

        let attachmentText;
        if (attachment.text) {
            attachmentText = messageHtmlToComponent(formatText(attachment.text));
        }

        let useBorderStyle;
        if (attachment.color && attachment.color[0] === '#') {
            useBorderStyle = {borderLeftColor: attachment.color};
        }

        return (
            <div
                className={'attachment'}
                ref='attachment'
            >
                <div className='attachment__content'>
                    <div
                        className={useBorderStyle ? 'clearfix attachment__container' : 'clearfix attachment__container attachment__container--' + attachment.color}
                        style={useBorderStyle}
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
