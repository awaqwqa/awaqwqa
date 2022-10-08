from django.shortcuts import render
 
def runoob(request):
    context          = {}
    context['hello'] = 'test'
    return render(request, 'about.html', context)