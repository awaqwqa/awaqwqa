from django.shortcuts import render
 
def runoob(request):
    context          = {}
    context['hello'] = 'test'
    return render(request, 'runoob.html', context)