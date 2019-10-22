import React from 'react';
import PropTypes from 'prop-types';

import {ActionButtonType} from 'utils/constants';

import ActionButton from './action_button';

export default class ActionView extends React.PureComponent {
    static propTypes = {
        post: PropTypes.object.isRequired,
        attachment: PropTypes.object.isRequired,
        pollMetadata: PropTypes.object,
        siteUrl: PropTypes.string.isRequired,

        actions: PropTypes.shape({
            fetchPollMetadata: PropTypes.func.isRequired,
        }).isRequired,
    }

    componentDidMount() {
        this.props.actions.fetchPollMetadata(this.props.siteUrl, this.props.post.props.poll_id);
    }

    render() {
        const actions = this.props.attachment.actions;
        if (!actions || !actions.length) {
            return '';
        }

        const content = [];
        const adminContent = [];
        const metadataMap = this.props.pollMetadata || {};
        const metadata = metadataMap[this.props.post.props.poll_id] || {};

        actions.
            filter((action) => action.id && action.name).
            forEach((action) => {
                switch (action.type) {
                case ActionButtonType.BUTTON:
                    content.push(
                        <ActionButton
                            key={action.id}
                            action={action}
                            postId={this.props.post.id}
                            hasVoted={metadata.voted_answers && (metadata.voted_answers.indexOf(action.name) >= 0)}
                        />
                    );
                    break;
                case ActionButtonType.MATTERPOLL_ADMIN_BUTTON:
                    if (metadata.admin_permission) {
                        adminContent.push(
                            <ActionButton
                                key={action.id}
                                action={action}
                                postId={this.props.post.id}
                                hasVoted={false}
                            />
                        );
                    }
                    break;
                default:
                    break;
                }
            });

        return (
            <div>
                <div className='attachment-actions'>
                    {content}
                </div>
                <div className='attachment-actions'>
                    {adminContent}
                </div>
            </div>
        );
    }
}
