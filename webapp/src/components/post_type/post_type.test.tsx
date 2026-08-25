import React from 'react';
import {render, screen} from '@testing-library/react';

import PostType from '@/components/post_type/post_type';

import type {Attachment} from '@/types/poll';

// ActionView is connected to the store; stand in for it so these tests cover
// PostType's own rendering of the attachment.
jest.mock('@/components/post_type/action_view', () => ({
    __esModule: true,
    default: ({attachment}: {attachment: Attachment}) => (
        <div data-testid='action-view'>{attachment.title}</div>
    ),
}));

describe('components/post_type/PostType', () => {
    const attachment: Attachment = {
        author_name: 'sample_name',
        title: 'sample_title',
        text: 'sample_text',
    };

    const propsWithAttachments = (attachments: Attachment[]) => ({
        post: {
            id: 'post_id',
            props: {attachments},
        },
        theme: {},
        options: {},
        postTypeComponentId: '',
    });

    test('should render the author, title and text of the first attachment', () => {
        const {asFragment} = render(<PostType {...propsWithAttachments([attachment])}/>);

        expect(screen.getByText('sample_name')).toBeInTheDocument();
        expect(screen.getByText('mockMessageHtmlToComponent(mockFormatText(sample_title))')).toBeInTheDocument();
        expect(screen.getByText('mockMessageHtmlToComponent(mockFormatText(sample_text))')).toBeInTheDocument();
        expect(asFragment()).toMatchSnapshot();
    });

    test('should still render the attachment shell without any attachments', () => {
        const {container} = render(<PostType {...propsWithAttachments([])}/>);

        expect(container.querySelector('.attachment')).toBeInTheDocument();
        expect(screen.queryByText('sample_name')).not.toBeInTheDocument();
    });

    test('should render only the first of several attachments', () => {
        render(
            <PostType
                {...propsWithAttachments([
                    attachment,
                    {author_name: 'second_name', title: 'second_title', text: 'second_text'},
                ])}
            />,
        );

        expect(screen.getByText('sample_name')).toBeInTheDocument();
        expect(screen.queryByText('second_name')).not.toBeInTheDocument();
    });
});
